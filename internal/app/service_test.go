package app

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/usingcoding/ttsh/internal/domain"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type memoryRepository struct{ sheets map[string]domain.Sheet }
type memoryHandle struct {
	repository *memoryRepository
	sheet      domain.Sheet
}

func (r *memoryRepository) Acquire(_ context.Context, date time.Time) (domain.SheetHandle, error) {
	sheet, ok := r.sheets[date.Format(time.DateOnly)]
	if !ok {
		sheet = domain.Sheet{Date: date}
	}
	return &memoryHandle{repository: r, sheet: sheet}, nil
}
func (r *memoryRepository) ListDates(_ context.Context, _ time.Time) ([]time.Time, error) {
	return nil, nil
}
func (h *memoryHandle) Sheet() domain.Sheet { return h.sheet }
func (h *memoryHandle) Save(sheet domain.Sheet) error {
	h.sheet = sheet
	h.repository.sheets[sheet.Date.Format(time.DateOnly)] = sheet
	return nil
}
func (h *memoryHandle) Close() error { return nil }

func TestSessionActivePoliciesAndDurations(t *testing.T) {
	now := time.Date(2026, 9, 29, 16, 22, 0, 0, time.Local)
	repository := &memoryRepository{sheets: map[string]domain.Sheet{}}
	service := New(repository, fixedClock{now}, false)
	session, err := service.Open(context.Background(), now)
	require.NoError(t, err)
	start, _ := domain.ParseTimeOfDay("14:05")
	_, _, err = session.Add(domain.EntryInput{Name: "Old", Start: start}, RejectActive)
	require.NoError(t, err)
	_, _, err = session.Add(domain.EntryInput{Name: "New", Start: domain.TimeOfDayAt(now)}, RejectActive)
	require.ErrorAs(t, err, new(domain.ActiveEntryConflictError))
	require.Len(t, session.List(), 1)
	index, _, err := session.Add(domain.EntryInput{Name: "New", Start: domain.TimeOfDayAt(now)}, StopExistingActive)
	require.NoError(t, err)
	require.Equal(t, 2, index)
	require.Equal(t, "2h:17m", domain.FormatDuration(session.CompletedTotal()))
	require.Equal(t, "0h:00m", domain.FormatDuration(session.EntryDuration(session.List()[1])))
	_, err = session.Remove(3)
	require.ErrorAs(t, err, new(domain.MissingPositionError))
}

func TestParallelEntriesAllowed(t *testing.T) {
	now := time.Date(2026, 9, 29, 16, 22, 0, 0, time.Local)
	service := New(&memoryRepository{sheets: map[string]domain.Sheet{}}, fixedClock{now}, true)
	session, err := service.Open(context.Background(), now)
	require.NoError(t, err)
	_, _, err = session.Add(domain.EntryInput{Name: "one", Start: domain.TimeOfDayAt(now)}, RejectActive)
	require.NoError(t, err)
	_, _, err = session.Add(domain.EntryInput{Name: "two", Start: domain.TimeOfDayAt(now)}, RejectActive)
	require.NoError(t, err)
}

func TestSessionRejectsHistoricalActiveEntries(t *testing.T) {
	now := time.Date(2026, 9, 29, 16, 22, 0, 0, time.Local)
	service := New(&memoryRepository{sheets: map[string]domain.Sheet{}}, fixedClock{now}, false)
	session, err := service.Open(context.Background(), now.AddDate(0, 0, -1))
	require.NoError(t, err)
	_, _, err = session.Add(domain.EntryInput{Name: "work", Start: domain.TimeOfDayAt(now)}, RejectActive)
	require.EqualError(t, err, "active entries are only allowed on the current date")
}
