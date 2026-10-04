package storage

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/usingcoding/ttsh/internal/domain"
)

func TestStorePersistsAndLocksSheets(t *testing.T) {
	paths := Paths{SheetsDir: t.TempDir()}
	store := New(paths)
	date := time.Date(2026, 9, 29, 0, 0, 0, 0, time.Local)
	handle, err := store.Acquire(context.Background(), date)
	require.NoError(t, err)
	require.Empty(t, handle.Sheet().Entries)
	require.NoFileExists(t, paths.SheetPath(date))
	start, _ := domain.ParseTimeOfDay("09:00")
	sheet := handle.Sheet()
	sheet.Entries = append(sheet.Entries, domain.Entry{Name: "Work", Start: start})
	require.NoError(t, handle.Save(sheet))
	contents, err := os.ReadFile(paths.SheetPath(date))
	require.NoError(t, err)
	require.Contains(t, string(contents), "name: \"Work\"")
	require.Contains(t, string(contents), "end: null")
	_, err = store.Acquire(context.Background(), date)
	require.ErrorAs(t, err, new(domain.LockUnavailableError))
	require.NoError(t, handle.Close())

	handle, err = store.Acquire(context.Background(), date)
	require.NoError(t, err)
	require.Len(t, handle.Sheet().Entries, 1)
	require.NoError(t, handle.Save(domain.Sheet{Date: date}))
	require.NoError(t, handle.Close())
	require.FileExists(t, paths.SheetPath(date))
}
