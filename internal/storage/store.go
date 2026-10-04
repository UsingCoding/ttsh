package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/usingcoding/ttsh/internal/domain"
	"gopkg.in/yaml.v3"
)

type Store struct{ Paths Paths }

func New(paths Paths) *Store { return &Store{Paths: paths} }

func (s *Store) Acquire(ctx context.Context, date time.Time) (domain.SheetHandle, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	date = domain.DateOnly(date)
	if err := os.MkdirAll(s.Paths.SheetsDir, 0o755); err != nil {
		return nil, fmt.Errorf("create sheets directory: %w", err)
	}
	lock, err := os.OpenFile(s.Paths.LockPath(date), os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		_ = lock.Close()
		if errors.Is(err, syscall.EWOULDBLOCK) || errors.Is(err, syscall.EAGAIN) {
			return nil, domain.LockUnavailableError{Date: date}
		}
		return nil, fmt.Errorf("lock sheet: %w", err)
	}
	handle := &handle{store: s, date: date, lock: lock}
	if err := handle.writeLockMetadata(); err != nil {
		_ = handle.Close()
		return nil, err
	}
	sheet, err := s.load(date)
	if err != nil {
		_ = handle.Close()
		return nil, err
	}
	handle.sheet = sheet
	return handle, nil
}

func (s *Store) ListDates(ctx context.Context, month time.Time) ([]time.Time, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.Paths.SheetsDir)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list sheets: %w", err)
	}
	month = domain.DateOnly(month)
	var dates []time.Time
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".yaml") {
			continue
		}
		date, err := time.ParseInLocation(time.DateOnly, strings.TrimSuffix(name, ".yaml"), time.Local)
		if err == nil && date.Year() == month.Year() && date.Month() == month.Month() {
			dates = append(dates, date)
		}
	}
	return dates, nil
}

type yamlEntry struct {
	Name        string  `yaml:"name"`
	Start       string  `yaml:"start"`
	End         *string `yaml:"end"`
	Description string  `yaml:"description"`
}

func (entry yamlEntry) MarshalYAML() (any, error) {
	mapping := &yaml.Node{Kind: yaml.MappingNode}
	add := func(key, value string) {
		mapping.Content = append(mapping.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: key},
			&yaml.Node{Kind: yaml.ScalarNode, Value: value, Style: yaml.DoubleQuotedStyle})
	}
	add("name", entry.Name)
	add("start", entry.Start)
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: "end"})
	if entry.End == nil {
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!null", Value: "null"})
	} else {
		mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Value: *entry.End, Style: yaml.DoubleQuotedStyle})
	}
	add("description", entry.Description)
	return mapping, nil
}

func (s *Store) load(date time.Time) (domain.Sheet, error) {
	file, err := os.Open(s.Paths.SheetPath(date))
	if errors.Is(err, os.ErrNotExist) {
		return domain.Sheet{Date: date}, nil
	}
	if err != nil {
		return domain.Sheet{}, fmt.Errorf("open sheet: %w", err)
	}
	defer func() { _ = file.Close() }()
	contents, err := io.ReadAll(file)
	if err != nil {
		return domain.Sheet{}, fmt.Errorf("read sheet: %w", err)
	}
	var encoded []yamlEntry
	if err := yaml.Unmarshal(contents, &encoded); err != nil {
		return domain.Sheet{}, fmt.Errorf("decode sheet: %w", err)
	}
	sheet := domain.Sheet{Date: date, Entries: make([]domain.Entry, 0, len(encoded))}
	for _, item := range encoded {
		start, err := domain.ParseTimeOfDay(item.Start)
		if err != nil {
			return domain.Sheet{}, fmt.Errorf("decode sheet start: %w", err)
		}
		input := domain.EntryInput{Name: item.Name, Start: start, Description: item.Description}
		if item.End != nil {
			end, err := domain.ParseTimeOfDay(*item.End)
			if err != nil {
				return domain.Sheet{}, fmt.Errorf("decode sheet end: %w", err)
			}
			input.End = &end
		}
		if err := input.Validate(); err != nil {
			return domain.Sheet{}, fmt.Errorf("decode sheet entry: %w", err)
		}
		sheet.Entries = append(sheet.Entries, domain.Entry(input))
	}
	return sheet, nil
}

type handle struct {
	store  *Store
	date   time.Time
	lock   *os.File
	sheet  domain.Sheet
	closed bool
}

func (h *handle) Sheet() domain.Sheet { return h.sheet }

func (h *handle) Save(sheet domain.Sheet) error {
	if h.closed {
		return errors.New("sheet handle is closed")
	}
	encoded := make([]yamlEntry, 0, len(sheet.Entries))
	for _, entry := range sheet.Entries {
		item := yamlEntry{Name: entry.Name, Start: entry.Start.String(), Description: entry.Description}
		if entry.End != nil {
			end := entry.End.String()
			item.End = &end
		}
		encoded = append(encoded, item)
	}
	data, err := yaml.Marshal(encoded)
	if err != nil {
		return fmt.Errorf("encode sheet: %w", err)
	}
	temp, err := os.CreateTemp(h.store.Paths.SheetsDir, ".sheet-*.tmp")
	if err != nil {
		return fmt.Errorf("create sheet temp file: %w", err)
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if _, err := temp.Write(data); err != nil {
		_ = temp.Close()
		return fmt.Errorf("write sheet: %w", err)
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		return fmt.Errorf("sync sheet: %w", err)
	}
	if err := temp.Close(); err != nil {
		return fmt.Errorf("close sheet: %w", err)
	}
	if err := os.Rename(tempName, h.store.Paths.SheetPath(h.date)); err != nil {
		return fmt.Errorf("replace sheet: %w", err)
	}
	directory, err := os.Open(filepath.Dir(h.store.Paths.SheetPath(h.date)))
	if err != nil {
		return fmt.Errorf("open sheets directory: %w", err)
	}
	defer func() { _ = directory.Close() }()
	if err := directory.Sync(); err != nil {
		return fmt.Errorf("sync sheets directory: %w", err)
	}
	h.sheet = sheet
	return nil
}

func (h *handle) writeLockMetadata() error {
	host, err := os.Hostname()
	if err != nil {
		host = "unknown"
	}
	metadata := fmt.Sprintf("pid: %d\nhostname: %s\nstarted_at: %s\n", os.Getpid(), host, time.Now().Format(time.RFC3339))
	if err := h.lock.Truncate(0); err != nil {
		return fmt.Errorf("truncate lock metadata: %w", err)
	}
	if _, err := h.lock.Seek(0, 0); err != nil {
		return fmt.Errorf("seek lock metadata: %w", err)
	}
	if _, err := h.lock.WriteString(metadata); err != nil {
		return fmt.Errorf("write lock metadata: %w", err)
	}
	if err := h.lock.Sync(); err != nil {
		return fmt.Errorf("sync lock metadata: %w", err)
	}
	return nil
}

func (h *handle) Close() error {
	if h.closed {
		return nil
	}
	h.closed = true
	unlockErr := syscall.Flock(int(h.lock.Fd()), syscall.LOCK_UN)
	closeErr := h.lock.Close()
	if unlockErr != nil {
		return fmt.Errorf("unlock sheet: %w", unlockErr)
	}
	if closeErr != nil {
		return fmt.Errorf("close lock: %w", closeErr)
	}
	return nil
}
