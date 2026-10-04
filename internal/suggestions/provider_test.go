package suggestions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCommandParsesLineOutput(t *testing.T) {
	provider := Command{Command: []string{"sh", "-c", "printf ' A \\n\\nB\\n'"}, Timeout: time.Second}
	require.Equal(t, []string{"A", "B"}, provider.Names(context.Background()))
}
