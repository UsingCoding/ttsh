package domain

import (
	"fmt"
	"time"
)

type LockUnavailableError struct{ Date time.Time }

func (e LockUnavailableError) Error() string {
	return fmt.Sprintf("%s is currently locked by another ttsh process", DateOnly(e.Date).Format(time.DateOnly))
}

type MissingPositionError struct {
	Position int
	Date     time.Time
}

func (e MissingPositionError) Error() string {
	return fmt.Sprintf("entry #%d does not exist for %s", e.Position, DateOnly(e.Date).Format(time.DateOnly))
}

type ActiveEntryConflictError struct {
	Position int
	Entry    Entry
}

func (e ActiveEntryConflictError) Error() string {
	return fmt.Sprintf("cannot start entry: #%d <%s> is already active", e.Position, e.Entry.Name)
}

type ValidationError struct{ Message string }

func (e ValidationError) Error() string { return e.Message }
