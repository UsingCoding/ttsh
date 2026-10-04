package suggestions

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"strings"
	"time"
)

type Suggestion struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Provider interface {
	Suggestions(context.Context) []Suggestion
}

type None struct{}

func (None) Suggestions(context.Context) []Suggestion { return nil }

type Command struct {
	Command []string
	Format  string
	Timeout time.Duration
	Logger  *slog.Logger
}

func (p Command) Suggestions(ctx context.Context) []Suggestion {
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
	switch p.Format {
	case "", "lines":
		return parseLines(output)
	case "json":
		return p.parseJSON(output)
	default:
		p.logParseFailure(fmt.Errorf("unsupported suggestions format %q", p.Format))
		return nil
	}
}

func parseLines(output []byte) []Suggestion {
	lines := strings.Split(string(output), "\n")
	suggestions := make([]Suggestion, 0, len(lines))
	for _, line := range lines {
		if name := strings.TrimSpace(line); name != "" {
			suggestions = append(suggestions, Suggestion{Name: name})
		}
	}
	return suggestions
}

func (p Command) parseJSON(output []byte) []Suggestion {
	output = bytes.TrimSpace(output)
	if len(output) == 0 || output[0] != '[' {
		p.logParseFailure(fmt.Errorf("suggestion JSON must be an array"))
		return nil
	}
	var parsed []Suggestion
	if err := json.Unmarshal(output, &parsed); err != nil {
		p.logParseFailure(err)
		return nil
	}
	suggestions := make([]Suggestion, 0, len(parsed))
	for _, suggestion := range parsed {
		if name := strings.TrimSpace(suggestion.Name); name != "" {
			suggestions = append(suggestions, Suggestion{Name: name, Description: suggestion.Description})
		}
	}
	return suggestions
}

func (p Command) logParseFailure(err error) {
	if p.Logger != nil {
		p.Logger.Debug("parse suggestion output failed", "error", err)
	}
}
