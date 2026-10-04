package suggestions

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestCommandParsesLineOutput(t *testing.T) {
	provider := Command{Command: []string{"sh", "-c", "printf ' A \\n\\nB\\n'"}, Timeout: time.Second}
	require.Equal(t, []Suggestion{{Name: "A"}, {Name: "B"}}, provider.Suggestions(context.Background()))
}

func TestCommandParsesJSONOutput(t *testing.T) {
	provider := Command{
		Command: []string{"sh", "-c", `printf '%s' '[{"name":" A ","description":"first"},{"name":"  ","description":"ignored"},{"name":"B","description":"second"}]'`},
		Format:  "json",
		Timeout: time.Second,
	}

	require.Equal(t, []Suggestion{
		{Name: "A", Description: "first"},
		{Name: "B", Description: "second"},
	}, provider.Suggestions(context.Background()))
}

func TestCommandMalformedJSONReturnsNoSuggestions(t *testing.T) {
	provider := Command{
		Command: []string{"sh", "-c", "printf '['"},
		Format:  "json",
		Timeout: time.Second,
	}

	require.Empty(t, provider.Suggestions(context.Background()))
}
