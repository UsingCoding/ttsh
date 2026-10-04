package app

import (
	"context"
	"fmt"
	"time"

	"github.com/usingcoding/ttsh/internal/domain"
)

type Clock interface{ Now() time.Time }
type localClock struct{}

func (localClock) Now() time.Time { return time.Now() }

type ActiveEntryPolicy uint8

const (
	RejectActive ActiveEntryPolicy = iota
	StopExistingActive
)

type Service struct {
	repository           domain.SheetRepository
	clock                Clock
	allowParallelEntries bool
}

func New(repository domain.SheetRepository, clock Clock, allowParallelEntries bool) *Service {
	if clock == nil {
		clock = localClock{}
	}
	return &Service{repository: repository, clock: clock, allowParallelEntries: allowParallelEntries}
}

func (s *Service) Open(ctx context.Context, date time.Time) (*SheetSession, error) {
	handle, err := s.repository.Acquire(ctx, domain.DateOnly(date))
	if err != nil {
		return nil, err
	}
	return &SheetSession{handle: handle, sheet: handle.Sheet(), clock: s.clock, allowParallelEntries: s.allowParallelEntries}, nil
}

func (s *Service) DatesWithEntries(ctx context.Context, month time.Time) ([]time.Time, error) {
	return s.DatesWithEntriesForSession(ctx, month, nil)
}

func (s *Service) DatesWithEntriesForSession(ctx context.Context, month time.Time, held *SheetSession) ([]time.Time, error) {
	dates, err := s.repository.ListDates(ctx, month)
	if err != nil {
		return nil, err
	}
	result := make([]time.Time, 0, len(dates))
	for _, date := range dates {
		if held != nil && domain.SameDate(date, held.Date()) {
			if len(held.List()) > 0 {
				result = append(result, domain.DateOnly(date))
			}
			continue
		}
		session, err := s.Open(ctx, date)
		if err != nil {
			return nil, err
		}
		if len(session.List()) > 0 {
			result = append(result, domain.DateOnly(date))
		}
		if err := session.Close(); err != nil {
			return nil, err
		}
	}
	return result, nil
}

type SheetSession struct {
	handle               domain.SheetHandle
	sheet                domain.Sheet
	clock                Clock
	allowParallelEntries bool
	closed               bool
}

func (s *SheetSession) Date() time.Time      { return s.sheet.Date }
func (s *SheetSession) List() []domain.Entry { return append([]domain.Entry(nil), s.sheet.Entries...) }

func (s *SheetSession) Get(index int) (domain.Entry, error) {
	if index < 1 || index > len(s.sheet.Entries) {
		return domain.Entry{}, domain.MissingPositionError{Position: index, Date: s.sheet.Date}
	}
	return s.sheet.Entries[index-1], nil
}

func (s *SheetSession) Add(input domain.EntryInput, policy ActiveEntryPolicy) (int, domain.Entry, error) {
	if err := input.Validate(); err != nil {
		return 0, domain.Entry{}, err
	}
	if input.End == nil && !domain.SameDate(s.sheet.Date, s.clock.Now()) {
		return 0, domain.Entry{}, domain.ValidationError{Message: "active entries are only allowed on the current date"}
	}
	candidate := s.copySheet()
	if input.End == nil && !s.allowParallelEntries {
		for i, entry := range candidate.Entries {
			if entry.End != nil {
				continue
			}
			if policy == RejectActive {
				return 0, domain.Entry{}, domain.ActiveEntryConflictError{Position: i + 1, Entry: entry}
			}
			now := domain.TimeOfDayAt(s.clock.Now())
			if now < entry.Start {
				return 0, domain.Entry{}, domain.ValidationError{Message: "end must not be earlier than start"}
			}
			candidate.Entries[i].End = &now
			break
		}
	}
	entry := domain.Entry(input)
	candidate.Entries = append(candidate.Entries, entry)
	if err := s.persist(candidate); err != nil {
		return 0, domain.Entry{}, err
	}
	return len(candidate.Entries), entry, nil
}

func (s *SheetSession) Update(index int, input domain.EntryInput) (domain.Entry, error) {
	if _, err := s.Get(index); err != nil {
		return domain.Entry{}, err
	}
	if err := input.Validate(); err != nil {
		return domain.Entry{}, err
	}
	if input.End == nil && !domain.SameDate(s.sheet.Date, s.clock.Now()) {
		return domain.Entry{}, domain.ValidationError{Message: "active entries are only allowed on the current date"}
	}
	candidate := s.copySheet()
	candidate.Entries[index-1] = domain.Entry(input)
	if err := s.persist(candidate); err != nil {
		return domain.Entry{}, err
	}
	return candidate.Entries[index-1], nil
}

func (s *SheetSession) Remove(index int) (domain.Entry, error) {
	entry, err := s.Get(index)
	if err != nil {
		return domain.Entry{}, err
	}
	candidate := s.copySheet()
	candidate.Entries = append(candidate.Entries[:index-1], candidate.Entries[index:]...)
	if err := s.persist(candidate); err != nil {
		return domain.Entry{}, err
	}
	return entry, nil
}

func (s *SheetSession) Stop(index int) (domain.Entry, error) {
	entry, err := s.Get(index)
	if err != nil {
		return domain.Entry{}, err
	}
	if entry.End != nil {
		return entry, nil
	}
	if !domain.SameDate(s.sheet.Date, s.clock.Now()) {
		return domain.Entry{}, domain.ValidationError{Message: "cannot stop an active entry on a non-current sheet"}
	}
	now := domain.TimeOfDayAt(s.clock.Now())
	input := domain.EntryInput(entry)
	input.End = &now
	if err := input.Validate(); err != nil {
		return domain.Entry{}, err
	}
	candidate := s.copySheet()
	candidate.Entries[index-1] = domain.Entry(input)
	if err := s.persist(candidate); err != nil {
		return domain.Entry{}, err
	}
	return candidate.Entries[index-1], nil
}

func (s *SheetSession) EntryDuration(entry domain.Entry) time.Duration {
	return entry.Duration(domain.TimeOfDayAt(s.clock.Now()))
}
func (s *SheetSession) CompletedTotal() time.Duration {
	var total time.Duration
	for _, entry := range s.sheet.Entries {
		if entry.End != nil {
			total += entry.Duration(0)
		}
	}
	return total
}
func (s *SheetSession) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	return s.handle.Close()
}
func (s *SheetSession) copySheet() domain.Sheet {
	return domain.Sheet{Date: s.sheet.Date, Entries: append([]domain.Entry(nil), s.sheet.Entries...)}
}
func (s *SheetSession) persist(sheet domain.Sheet) error {
	if s.closed {
		return fmt.Errorf("sheet session is closed")
	}
	if err := s.handle.Save(sheet); err != nil {
		return err
	}
	s.sheet = sheet
	return nil
}
