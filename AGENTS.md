# Repository Guidelines

## Project Overview

`ttsh` is a Go daily time-tracking application with two adapters over the same YAML-backed service: an interactive Bubble Tea TUI and scriptable `list`, `view`, `add`, and `remove` commands. The product contract is in `spec/mvp-01/mvp1.md`; CLI engineering rules are in `spec/mvp-01/cli.md`.

## Architecture & Data Flow

Dependency direction is deliberate:

```text
cmd/ttsh → internal/cli → internal/app → internal/domain
                         ↘ config, storage, suggestions, tui
storage → domain ports
```

- `cmd/ttsh/main.go` is startup/exit handling only. It constructs `cli.New`, prints one returned error, and must not gain business logic.
- `internal/cli/root.go` is the composition root. It loads config, wires `storage.Store`, `app.Service`, suggestions, logging, and either a CLI command or TUI.
- `internal/domain` owns typed entries, time-of-day parsing, validation, display-position errors, and the `SheetRepository`/`SheetHandle` ports. Keep YAML, CLI, and Bubble Tea types out of this package.
- `internal/storage` resolves XDG paths, acquires exclusive per-day `flock` locks, translates YAML, and atomically replaces sheet files.
- `internal/app.Service` opens a locked `SheetSession`; all entry mutations go through that session and persist before success is returned. Both interfaces must use this API rather than duplicate rules.
- `internal/tui.Model` retains the selected day session and routes input through a modal state machine. The selected TUI index is zero-based; `SheetSession` APIs accept one-based display IDs.

Typical flow: command/TUI → `Service.Open(date)` → locked `SheetSession` → validate/mutate → `SheetHandle.Save` → atomic YAML replacement → `Close` unlocks the day.

## Key Directories

- `cmd/ttsh/` — minimal executable entry point.
- `internal/cli/` — urfave/cli v3 command tree and dependency composition.
- `internal/app/` — stateful sheet use cases and clock-injected duration/active-entry behavior.
- `internal/domain/` — entities, validation, typed errors, repository ports.
- `internal/storage/` — XDG paths, advisory locks, YAML, durable persistence.
- `internal/config/` — TOML config loading and defaults.
- `internal/suggestions/` — no-op and subprocess-backed free-text suggestions.
- `internal/tui/` — Bubble Tea model, modal workflows, rendering.
- `spec/mvp-01/` — authoritative product and CLI specifications.

## Development Commands

```sh
mise install
mise run build  # Preferred: GoReleaser snapshot build
mise run lint   # Preferred: configured static analysis
mise run test   # Preferred: full Go test suite
gofmt -w <go-files>
goimports -w <go-files>
go vet ./...
goreleaser check
```

Prefer `mise run build` and `mise run test` over direct `go build` and `go test` calls. Run locally with `go run ./cmd/ttsh`; no subcommand opens the TUI. Scriptable forms are:

```text
ttsh list [--date YYYY-MM-DD]
ttsh view <id> [--date YYYY-MM-DD]
ttsh add <name> [--date YYYY-MM-DD] [--start HH:MM] [--end HH:MM] [--description TEXT]
ttsh remove <id> [--date YYYY-MM-DD]
```

## Code Conventions & Common Patterns

- Format with both `gofmt` and `goimports`. Use standard-library facilities unless a declared dependency is needed.
- Keep `main.go` thin and `urfave/cli` usage inside `internal/cli`. Use typed CLI arguments and kebab-case flags.
- Use `domain.TimeOfDay`, `domain.EntryInput`, and typed errors (`MissingPositionError`, `ActiveEntryConflictError`, etc.); do not spread raw time strings or reimplement user-facing errors in adapters.
- Parse input at adapter boundaries, then delegate validation and mutations to `SheetSession`. Return errors upward; the executable prints them once.
- Inject dependencies instead of reaching into globals: `cli.Dependencies`, `app.New(repository, clock, allowParallelEntries)`, and provider/logger constructor inputs are established patterns.
- Respect lock ownership. CLI actions use short-lived sessions; the TUI owns one session until day switch/exit. Close every acquired session/handle on every path.
- Keep persistence adapter-only: YAML fields and `end: null` belong in `internal/storage`, while display `#N` is calculated from entry order and never persisted.
- Pass `context.Context` through repository and suggestion work. Bubble Tea async work is expressed as `tea.Cmd`/`tea.Tick`; keep modal state in `Model` rather than ad-hoc goroutines.

## Important Files

- `README.md` — build/run usage, XDG locations, suggestion configuration, and lock semantics.
- `spec/mvp-01/mvp1.md` — behavior, exact CLI output/error contract, YAML format, and scope boundaries.
- `internal/domain/entry.go` — time model, entry validation, durations, local-date helpers.
- `internal/domain/ports.go` — repository boundary.
- `internal/storage/store.go` — lock lifecycle, YAML conversion, atomic writes.
- `internal/app/service.go` — `Service`, `SheetSession`, active-entry policy, and mutation boundary.
- `internal/cli/root.go` — command definitions, output formatting, and composition.
- `internal/tui/model.go` — TUI state machine and held-session lifecycle.
- `mise.toml`, `.golangci.yml`, `.goreleaser.yaml`, `go.mod` — required tooling and release matrix.

## Runtime/Tooling Preferences

- Runtime: Go **1.25.0** (`go.mod`, `mise.toml`); use Go modules and commit `go.sum` changes with dependency updates.
- `mise` provisions Go, `goimports`, `golangci-lint`, and GoReleaser. The non-Go tools intentionally use `latest` in `mise.toml`.
- Use the described `mise run build`, `mise run lint`, and `mise run test` tasks for routine build, lint, and test work; direct Go commands remain appropriate for focused checks.
- GoReleaser builds `./cmd/ttsh` with `CGO_ENABLED=0` for Darwin/Linux on `amd64` and `arm64`, injecting `internal/version.Version`.
- There is no Makefile, CI workflow, Node/Bun runtime, or repository script directory.

## Testing & QA

- Tests live beside production files in the same package and use `testing` plus fail-fast `github.com/stretchr/testify/require`.
- Follow existing test setup: fixed clocks and inline `SheetRepository` fakes in `internal/app/service_test.go`; real `storage.Store` plus `t.TempDir()` for storage/CLI/TUI integration paths; injected CLI dependencies and `bytes.Buffer` output capture in `internal/cli/root_test.go`.
- Drive TUI behavior directly with `tea.KeyMsg` and assert model state in `internal/tui/model_test.go`. Keep behavior tests deterministic; use a fixed clock when time affects expectations.
- Preserve exact user-visible CLI output and typed-error behavior. Run `mise run test` and `mise run lint` for non-trivial changes; use `goreleaser check` when changing release config.
