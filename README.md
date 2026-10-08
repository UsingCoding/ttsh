# ttsh

`ttsh` is a keyboard-driven daily time-sheet TUI with scriptable commands over the same local YAML sheets.

## Build and run

```sh
mise run build
mise run test
mise run lint
```

Running `ttsh` with no subcommand opens the TUI. CLI forms:

```text
ttsh list [--date YYYY-MM-DD] [--json]
ttsh view <id> [--date YYYY-MM-DD] [--json]
ttsh add <name> [--date YYYY-MM-DD] [--start HH:MM] [--end HH:MM] [--description TEXT] [--json]
ttsh remove <id> [--date YYYY-MM-DD] [--json]
```

## JSON API

Every scriptable command accepts a per-command `--json` flag. Successful `list --json` responses are a JSON array; successful `view`, `add`, and `remove` responses are one JSON entry object. Every successful response is newline-terminated.

Each entry always has `id` (display ID), `date` (`YYYY-MM-DD`), `name`, `start` (`HH:MM`), `end` (`HH:MM` or `null` for an active entry), `description` (including `""`), and `duration` (for example, `0h:45m`). An empty JSON list is `[]`. JSON mode does not wrap responses or change failures: errors remain on stderr with a non-zero exit status.

State lives in `$XDG_STATE_HOME/ttsh/sheets` or, when unset, `$HOME/.local/state/ttsh/sheets`. Configuration is `$XDG_CONFIG_HOME/ttsh/config.toml` or `$HOME/.config/ttsh/config.toml`.

Suggestions are optional command output. `format = "lines"` is the default:

```toml
[suggestions]
enabled = true
command = ["some-command", "list-current-work"]
format = "lines"
timeout = "2s"
```

Each non-empty trimmed output line is an entry name; lines cannot provide a description:

```text
YT-517
YT-530
```

Use `format = "json"` when suggestions need explanatory text:

```toml
[suggestions]
enabled = true
command = ["some-command", "list-current-work"]
format = "json"
timeout = "2s"
```

The command must emit one JSON array. Each object has a required `name` and an optional `description`:

```json
[
  {"name": "YT-517", "description": "Fix account service"}
]
```

The TUI displays a suggestion's description beside its name, but accepting it fills and persists only the name. Provider failures leave manual entry available.

Each selected day holds an exclusive per-day advisory lock. A TUI retains that lock until it switches days or exits; CLI commands retain it only for their operation.
