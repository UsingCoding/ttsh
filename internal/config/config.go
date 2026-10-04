package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

const defaultSuggestionTimeout = 2 * time.Second

type Suggestions struct {
	Enabled bool
	Command []string
	Format  string
	Timeout time.Duration
}

type Config struct {
	AllowParallelEntries bool `toml:"allow_parallel_entries"`
	Suggestions          Suggestions
}

type fileConfig struct {
	AllowParallelEntries bool `toml:"allow_parallel_entries"`
	Suggestions          struct {
		Enabled bool
		Command []string
		Format  string
		Timeout string
	}
}

func Path() (string, error) {
	if root := os.Getenv("XDG_CONFIG_HOME"); root != "" {
		return filepath.Join(root, "ttsh", "config.toml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".config", "ttsh", "config.toml"), nil
}

func Load() (Config, error) {
	path, err := Path()
	if err != nil {
		return Config{}, err
	}
	return LoadPath(path)
}

func LoadPath(path string) (Config, error) {
	cfg := Config{Suggestions: Suggestions{Format: "lines", Timeout: defaultSuggestionTimeout}}
	var raw fileConfig
	if _, err := toml.DecodeFile(path, &raw); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, fmt.Errorf("load config: %w", err)
	}
	cfg.AllowParallelEntries = raw.AllowParallelEntries
	cfg.Suggestions.Enabled = raw.Suggestions.Enabled
	cfg.Suggestions.Command = raw.Suggestions.Command
	cfg.Suggestions.Format = raw.Suggestions.Format
	if cfg.Suggestions.Format == "" {
		cfg.Suggestions.Format = "lines"
	}
	if cfg.Suggestions.Format != "lines" && cfg.Suggestions.Format != "json" {
		return Config{}, fmt.Errorf("unsupported suggestions format %q", cfg.Suggestions.Format)
	}
	if raw.Suggestions.Timeout != "" {
		timeout, err := time.ParseDuration(raw.Suggestions.Timeout)
		if err != nil || timeout <= 0 {
			return Config{}, fmt.Errorf("invalid suggestions timeout %q", raw.Suggestions.Timeout)
		}
		cfg.Suggestions.Timeout = timeout
	}
	return cfg, nil
}
