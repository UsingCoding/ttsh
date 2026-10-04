package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestParseTimeOfDayStrict(t *testing.T) {
	value, err := ParseTimeOfDay("09:30")
	require.NoError(t, err)
	require.Equal(t, "09:30", value.String())
	for _, invalid := range []string{"9:30", "24:00", "09:60", "0900"} {
		_, err := ParseTimeOfDay(invalid)
		require.Error(t, err, invalid)
	}
}

func TestEntryInputValidation(t *testing.T) {
	start, _ := ParseTimeOfDay("10:00")
	end, _ := ParseTimeOfDay("09:59")
	require.Error(t, (EntryInput{Start: start}).Validate())
	require.Error(t, (EntryInput{Name: "work", Start: start, End: &end}).Validate())
	equal := start
	require.NoError(t, (EntryInput{Name: "work", Start: start, End: &equal}).Validate())
}

func TestEntryDuration(t *testing.T) {
	start, _ := ParseTimeOfDay("09:00")
	end, _ := ParseTimeOfDay("09:45")
	require.Equal(t, 45*time.Minute, (Entry{Start: start, End: &end}).Duration(start))
	now, _ := ParseTimeOfDay("10:15")
	require.Equal(t, "1h:15m", FormatDuration((Entry{Start: start}).Duration(now)))
}
