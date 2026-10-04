package domain

import (
	"fmt"
	"regexp"
	"time"
)

var timeOfDayPattern = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

type TimeOfDay uint16

func ParseTimeOfDay(value string) (TimeOfDay, error) {
	if !timeOfDayPattern.MatchString(value) {
		return 0, fmt.Errorf("invalid time %q: expected HH:MM", value)
	}
	var hour, minute int
	_, err := fmt.Sscanf(value, "%02d:%02d", &hour, &minute)
	if err != nil {
		return 0, fmt.Errorf("parse time %q: %w", value, err)
	}
	return TimeOfDay(hour*60 + minute), nil
}

func (t TimeOfDay) String() string {
	return fmt.Sprintf("%02d:%02d", int(t)/60, int(t)%60)
}

func (t TimeOfDay) DurationUntil(end TimeOfDay) time.Duration {
	return time.Duration(int(end)-int(t)) * time.Minute
}

func TimeOfDayAt(value time.Time) TimeOfDay {
	return TimeOfDay(value.Hour()*60 + value.Minute())
}

type Entry struct {
	Name        string
	Start       TimeOfDay
	End         *TimeOfDay
	Description string
}

type EntryInput Entry

type Sheet struct {
	Date    time.Time
	Entries []Entry
}

func (input EntryInput) Validate() error {
	if input.Name == "" {
		return ValidationError{Message: "name is required"}
	}
	if input.End != nil && *input.End < input.Start {
		return ValidationError{Message: "end must not be earlier than start"}
	}
	return nil
}

func (entry Entry) Duration(now TimeOfDay) time.Duration {
	if entry.End != nil {
		return entry.Start.DurationUntil(*entry.End)
	}
	return entry.Start.DurationUntil(now)
}

func FormatDuration(value time.Duration) string {
	minutes := int(value / time.Minute)
	if minutes < 0 {
		minutes = 0
	}
	return fmt.Sprintf("%dh:%02dm", minutes/60, minutes%60)
}

func SameDate(left, right time.Time) bool {
	left = left.In(time.Local)
	right = right.In(time.Local)
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}

func DateOnly(value time.Time) time.Time {
	local := value.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}
