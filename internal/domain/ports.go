package domain

import (
	"context"
	"time"
)

type SheetRepository interface {
	Acquire(context.Context, time.Time) (SheetHandle, error)
	ListDates(context.Context, time.Time) ([]time.Time, error)
}

type SheetHandle interface {
	Sheet() Sheet
	Save(Sheet) error
	Close() error
}
