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

Suggestions are optional line-based command output:

```toml
[suggestions]
enabled = true
command = ["some-command", "list-current-work"]
format = "lines"
timeout = "2s"
```

Each selected day holds an exclusive per-day advisory lock. A TUI retains that lock until it switches days or exits; CLI commands retain it only for their operation.
