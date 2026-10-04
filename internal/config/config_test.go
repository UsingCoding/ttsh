package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLoadPathDefaultsAndValidation(t *testing.T) {
	cfg, err := LoadPath(filepath.Join(t.TempDir(), "missing.toml"))
	require.NoError(t, err)
	require.False(t, cfg.AllowParallelEntries)
	require.Equal(t, 2*time.Second, cfg.Suggestions.Timeout)
	path := filepath.Join(t.TempDir(), "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[suggestions]\nenabled = true\nformat = \"json\"\n"), 0o600))
	_, err = LoadPath(path)
	require.Error(t, err)
}
