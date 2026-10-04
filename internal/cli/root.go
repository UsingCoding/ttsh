package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"

	urfave "github.com/urfave/cli/v3"
	"github.com/usingcoding/ttsh/internal/app"
	"github.com/usingcoding/ttsh/internal/config"
	"github.com/usingcoding/ttsh/internal/domain"
	"github.com/usingcoding/ttsh/internal/storage"
	"github.com/usingcoding/ttsh/internal/suggestions"
	"github.com/usingcoding/ttsh/internal/tui"
	"github.com/usingcoding/ttsh/internal/version"
)

type Dependencies struct {
	Service  *app.Service
	Clock    app.Clock
	Provider suggestions.Provider
	Logger   *slog.Logger
	RunTUI   func(*app.Service, suggestions.Provider, *slog.Logger) error
}

type dependencies struct {
	service  *app.Service
	provider suggestions.Provider
	logger   *slog.Logger
	runTUI   func(*app.Service, suggestions.Provider, *slog.Logger) error
}

func New(input Dependencies) *urfave.Command {
	deps := dependencies{service: input.Service, provider: input.Provider, logger: input.Logger, runTUI: input.RunTUI}
	if deps.logger == nil {
		deps.logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	if deps.runTUI == nil {
		deps.runTUI = tui.Run
	}
	return &urfave.Command{Name: "ttsh", Usage: "daily time tracking", Version: version.Version, Action: func(ctx context.Context, _ *urfave.Command) error {
		if err := deps.initialize(input.Clock); err != nil {
			return err
		}
		return deps.runTUI(deps.service, deps.provider, deps.logger)
	}, Commands: []*urfave.Command{newListCommand(&deps, input.Clock), newViewCommand(&deps, input.Clock), newAddCommand(&deps, input.Clock), newRemoveCommand(&deps, input.Clock)}}
}

func (d *dependencies) initialize(clock app.Clock) error {
	if d.service != nil {
		if d.provider == nil {
			d.provider = suggestions.None{}
		}
		return nil
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	paths, err := storage.NewPaths()
	if err != nil {
		return err
	}
	d.service = app.New(storage.New(paths), clock, cfg.AllowParallelEntries)
	if cfg.Suggestions.Enabled && len(cfg.Suggestions.Command) > 0 {
		d.provider = suggestions.Command{Command: cfg.Suggestions.Command, Format: cfg.Suggestions.Format, Timeout: cfg.Suggestions.Timeout, Logger: d.logger}
	} else {
		d.provider = suggestions.None{}
	}
	return nil
}

func newListCommand(deps *dependencies, clock app.Clock) *urfave.Command {
	return &urfave.Command{Name: "list", Usage: "list entries", Flags: dateFlags(), Action: func(ctx context.Context, cmd *urfave.Command) error {
		if err := deps.initialize(clock); err != nil {
			return err
		}
		date, err := commandDate(cmd, clock)
		if err != nil {
			return err
		}
		session, err := deps.service.Open(ctx, date)
		if err != nil {
			return err
		}
		defer func() { _ = session.Close() }()
		entries := session.List()
		if len(entries) == 0 {
			_, err = fmt.Fprintf(output(cmd), "No entries for %s.\n", date.Format(time.DateOnly))
			return err
		}
		for i, entry := range entries {
			end := "..."
			if entry.End != nil {
				end = entry.End.String()
			}
			line := fmt.Sprintf("#%d <%s> @%s -> %s", i+1, entry.Name, entry.Start, end)
			if entry.Description != "" {
				line += " # " + entry.Description
			}
			if _, err = fmt.Fprintln(output(cmd), line); err != nil {
				return err
			}
		}
		return nil
	}}
}
func newViewCommand(deps *dependencies, clock app.Clock) *urfave.Command {
	return &urfave.Command{Name: "view", Usage: "view an entry", Arguments: []urfave.Argument{&urfave.StringArg{Name: "id"}}, Flags: dateFlags(), Action: func(ctx context.Context, cmd *urfave.Command) error {
		if err := deps.initialize(clock); err != nil {
			return err
		}
		date, err := commandDate(cmd, clock)
		if err != nil {
			return err
		}
		index, err := parseID(cmd.StringArg("id"))
		if err != nil {
			return err
		}
		session, err := deps.service.Open(ctx, date)
		if err != nil {
			return err
		}
		defer func() { _ = session.Close() }()
		entry, err := session.Get(index)
		if err != nil {
			return err
		}
		end := "-"
		if entry.End != nil {
			end = entry.End.String()
		}
		_, err = fmt.Fprintf(output(cmd), "ID:          #%d\nName:        %s\nStart:       %s\nEnd:         %s\nSpent:       %s\nDescription: %s\n", index, entry.Name, entry.Start, end, domain.FormatDuration(session.EntryDuration(entry)), entry.Description)
		return err
	}}
}
func newAddCommand(deps *dependencies, clock app.Clock) *urfave.Command {
	return &urfave.Command{Name: "add", Usage: "add an entry", Arguments: []urfave.Argument{&urfave.StringArg{Name: "name"}}, Flags: append(dateFlags(), &urfave.StringFlag{Name: "start"}, &urfave.StringFlag{Name: "end"}, &urfave.StringFlag{Name: "description"}), Action: func(ctx context.Context, cmd *urfave.Command) error {
		if err := deps.initialize(clock); err != nil {
			return err
		}
		date, err := commandDate(cmd, clock)
		if err != nil {
			return err
		}
		name := cmd.StringArg("name")
		if name == "" {
			return domain.ValidationError{Message: "name is required"}
		}
		start := domain.TimeOfDayAt(now(clock))
		if value := cmd.String("start"); value != "" {
			start, err = domain.ParseTimeOfDay(value)
			if err != nil {
				return err
			}
		}
		input := domain.EntryInput{Name: name, Start: start, Description: cmd.String("description")}
		if value := cmd.String("end"); value != "" {
			end, err := domain.ParseTimeOfDay(value)
			if err != nil {
				return err
			}
			input.End = &end
		}
		session, err := deps.service.Open(ctx, date)
		if err != nil {
			return err
		}
		defer func() { _ = session.Close() }()
		index, entry, err := session.Add(input, app.RejectActive)
		if err != nil {
			return err
		}
		end := "..."
		if entry.End != nil {
			end = entry.End.String()
		}
		line := fmt.Sprintf("Added #%d <%s> @%s -> %s", index, entry.Name, entry.Start, end)
		if entry.Description != "" {
			line += " # " + entry.Description
		}
		_, err = fmt.Fprintln(output(cmd), line)
		return err
	}}
}
func newRemoveCommand(deps *dependencies, clock app.Clock) *urfave.Command {
	return &urfave.Command{Name: "remove", Usage: "remove an entry", Arguments: []urfave.Argument{&urfave.StringArg{Name: "id"}}, Flags: dateFlags(), Action: func(ctx context.Context, cmd *urfave.Command) error {
		if err := deps.initialize(clock); err != nil {
			return err
		}
		date, err := commandDate(cmd, clock)
		if err != nil {
			return err
		}
		index, err := parseID(cmd.StringArg("id"))
		if err != nil {
			return err
		}
		session, err := deps.service.Open(ctx, date)
		if err != nil {
			return err
		}
		defer func() { _ = session.Close() }()
		entry, err := session.Remove(index)
		if err != nil {
			return err
		}
		end := "..."
		if entry.End != nil {
			end = entry.End.String()
		}
		_, err = fmt.Fprintf(output(cmd), "Removed #%d <%s> @%s -> %s\n", index, entry.Name, entry.Start, end)
		return err
	}}
}

func output(cmd *urfave.Command) io.Writer {
	if cmd.Root().Writer != nil {
		return cmd.Root().Writer
	}
	return os.Stdout
}
func dateFlags() []urfave.Flag {
	return []urfave.Flag{&urfave.StringFlag{Name: "date", Usage: "sheet date (YYYY-MM-DD)"}}
}
func commandDate(cmd *urfave.Command, clock app.Clock) (time.Time, error) {
	value := cmd.String("date")
	if value == "" {
		return domain.DateOnly(now(clock)), nil
	}
	if len(value) != len(time.DateOnly) {
		return time.Time{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", value)
	}
	date, err := time.ParseInLocation(time.DateOnly, value, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date %q: expected YYYY-MM-DD", value)
	}
	return date, nil
}
func parseID(value string) (int, error) {
	value = strings.TrimPrefix(value, "#")
	index, err := strconv.Atoi(value)
	if err != nil || index < 1 {
		return 0, fmt.Errorf("invalid entry ID %q", value)
	}
	return index, nil
}
func now(clock app.Clock) time.Time {
	if clock != nil {
		return clock.Now()
	}
	return time.Now()
}
