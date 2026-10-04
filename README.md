# ttsh

`ttsh` is a keyboard-driven daily time-sheet TUI with scriptable commands over the same local YAML sheets.

## Build and run

```sh
go build ./cmd/ttsh
go run ./cmd/ttsh
```

Running `ttsh` with no subcommand opens the TUI. CLI forms:

```text
ttsh list [--date YYYY-MM-DD]
ttsh view <id> [--date YYYY-MM-DD]
ttsh add <name> [--date YYYY-MM-DD] [--start HH:MM] [--end HH:MM] [--description TEXT]
ttsh remove <id> [--date YYYY-MM-DD]
```

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
