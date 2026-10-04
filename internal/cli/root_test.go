package cli

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/usingcoding/ttsh/internal/app"
	"github.com/usingcoding/ttsh/internal/storage"
)

type testClock struct{ value time.Time }

func (c testClock) Now() time.Time { return c.value }

func TestCLICommandsUseSharedService(t *testing.T) {
	clock := testClock{value: time.Date(2026, 9, 29, 16, 0, 0, 0, time.Local)}
	service := app.New(storage.New(storage.Paths{SheetsDir: t.TempDir()}), clock, false)
	run := func(args ...string) (string, error) {
		var output bytes.Buffer
		command := New(Dependencies{Service: service, Clock: clock})
		command.Writer = &output
		err := command.Run(context.Background(), append([]string{"ttsh"}, args...))
		return output.String(), err
	}
	output, err := run("add", "Meeting", "--date", "2026-09-29", "--start", "13:00", "--end", "13:45", "--description", "Architecture sync")
	require.NoError(t, err)
	require.Equal(t, "Added #1 <Meeting> @13:00 -> 13:45 # Architecture sync\n", output)
	output, err = run("list", "--date", "2026-09-29")
	require.NoError(t, err)
	require.Equal(t, "#1 <Meeting> @13:00 -> 13:45 # Architecture sync\n", output)
	output, err = run("view", "#1", "--date", "2026-09-29")
	require.NoError(t, err)
	require.Contains(t, output, "ID:          #1\nName:        Meeting\n")
	require.Contains(t, output, "Spent:       0h:45m\n")
	output, err = run("remove", "1", "--date", "2026-09-29")
	require.NoError(t, err)
	require.Equal(t, "Removed #1 <Meeting> @13:00 -> 13:45\n", output)
	output, err = run("list", "--date", "2026-09-29")
	require.NoError(t, err)
	require.Equal(t, "No entries for 2026-09-29.\n", output)
	_, err = run("view", "9", "--date", "2026-09-29")
	require.EqualError(t, err, "entry #9 does not exist for 2026-09-29")
	_, err = run("list", "--date", "2026-9-29")
	require.EqualError(t, err, "invalid date \"2026-9-29\": expected YYYY-MM-DD")
	_, err = run("add", "First", "--date", "2026-09-29")
	require.NoError(t, err)
	_, err = run("add", "Second", "--date", "2026-09-29")
	require.EqualError(t, err, "cannot start entry: #1 <First> is already active")
}
