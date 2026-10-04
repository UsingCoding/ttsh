package suggestions

import (
	"context"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

type Provider interface {
	Names(context.Context) []string
}

type None struct{}

func (None) Names(context.Context) []string { return nil }

type Command struct {
	Command []string
	Timeout time.Duration
	Logger  *slog.Logger
}

func (p Command) Names(ctx context.Context) []string {
	if len(p.Command) == 0 {
		return nil
	}
	if p.Timeout <= 0 {
		p.Timeout = 2 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, p.Timeout)
	defer cancel()
	output, err := exec.CommandContext(ctx, p.Command[0], p.Command[1:]...).Output()
	if err != nil {
		if p.Logger != nil {
			p.Logger.Debug("suggestion command failed", "error", err)
		}
		return nil
	}
	lines := strings.Split(string(output), "\n")
	names := make([]string, 0, len(lines))
	for _, line := range lines {
		if name := strings.TrimSpace(line); name != "" {
			names = append(names, name)
		}
	}
	return names
}
