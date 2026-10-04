package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Paths struct{ SheetsDir string }

func NewPaths() (Paths, error) {
	if root := os.Getenv("XDG_STATE_HOME"); root != "" {
		return Paths{SheetsDir: filepath.Join(root, "ttsh", "sheets")}, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return Paths{}, fmt.Errorf("resolve home directory: %w", err)
	}
	return Paths{SheetsDir: filepath.Join(home, ".local", "state", "ttsh", "sheets")}, nil
}

func (p Paths) SheetPath(date time.Time) string {
	return filepath.Join(p.SheetsDir, date.In(time.Local).Format(time.DateOnly)+".yaml")
}

func (p Paths) LockPath(date time.Time) string {
	return filepath.Join(p.SheetsDir, date.In(time.Local).Format(time.DateOnly)+".lock")
}
