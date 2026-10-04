# Default Go CLI Application Specification

## 1. Purpose

This document defines the default architecture, technology stack, project structure, conventions, and engineering requirements for Go CLI applications.

It is intended to be reusable across projects ranging from small API clients to larger workflow/orchestration tools.

Primary goals:

- predictable project structure;
- thin CLI layer;
- business logic independent from CLI/framework code;
- easy unit testing;
- good scripting support;
- good interactive terminal UX when required;
- minimal dependencies;
- explicit dependency direction;
- easy local development using `mise`;
- simple static binary distribution;
- straightforward release through GoReleaser.

The CLI framework MUST be:

```text
github.com/urfave/cli/v3
```

Do not use Cobra unless a project explicitly overrides this specification.

---

# 2. Default Technology Stack

## Required

| Purpose | Technology |
|---|---|
| Language | Go |
| CLI commands / flags / arguments | `github.com/urfave/cli/v3` |
| Logging | `log/slog` |
| Configuration | `github.com/BurntSushi/toml` |
| Testing | `github.com/stretchr/testify` |
| HTTP | Go standard library `net/http` |
| JSON | Go standard library `encoding/json` |
| Tool / task management | `mise` |
| Linting | `golangci-lint` |
| Releases | GoReleaser |
| Formatting | `gofmt` / `goimports` |

Prefer the Go standard library whenever it provides a reasonable implementation.

Do not introduce libraries merely to wrap simple standard-library functionality.

---

# 3. Optional Stack

Dependencies should only be added when the feature actually exists.

## Interactive prompts

For simple questions, confirmations, selections, and forms:

```text
github.com/manifoldco/promptui
```

Examples:

- choose an item;
- request missing credentials;
- confirm a destructive operation;
- confirm an execution plan.

Interactive prompts MUST live outside the application/domain layer.

---

## Full TUI

For applications where the terminal UI is a primary interface:

```text
github.com/charmbracelet/bubbletea
github.com/charmbracelet/bubbles
github.com/charmbracelet/lipgloss
```

The TUI MUST call the same application services as regular CLI commands.

Do not duplicate business logic between:

```text
CLI
TUI
```

---

# 4. Project Structure

Default structure:

```text
.
├── cmd/
│   └── <app>/
│       └── main.go
│
├── internal/
│   ├── cli/
│   │   ├── root.go
│   │   ├── globals.go
│   │   ├── errors.go
│   │   └── <resource>/
│   │       ├── command.go
│   │       ├── list.go
│   │       ├── get.go
│   │       ├── create.go
│   │       └── delete.go
│   │
│   ├── app/
│   │   ├── app.go
│   │   └── <usecase>.go
│   │
│   ├── domain/
│   │   ├── <entity>.go
│   │   ├── errors.go
│   │   └── ports.go
│   │
│   ├── config/
│   │   ├── config.go
│   │   └── load.go
│   │
│   ├── output/
│   │   ├── output.go
│   │   ├── text.go
│   │   └── json.go
│   │
│   ├── version/
│   │   └── version.go
│   │
│   └── <adapter>/
│       ├── client.go
│       ├── models.go
│       └── ...
│
├── testdata/
│
├── spec/
│
├── mise.toml
├── .golangci.yml
├── .goreleaser.yaml
├── go.mod
├── go.sum
├── README.md
└── LICENSE
```

Not every directory has to exist.

Create packages only when there is actual code belonging to them.

For a small CLI, this is perfectly acceptable:

```text
cmd/app/main.go
internal/cli/
internal/app/
internal/config/
```

Do not create empty architectural layers simply to satisfy the directory structure.

---

# 5. Dependency Direction

The default dependency direction is:

```text
cmd
 ↓
cli
 ↓
app
 ↓
domain
 ↑
adapters
```

More precisely:

```text
cmd -> cli
cli -> app
app -> domain interfaces
adapters -> domain
```

The following MUST NOT happen:

```text
domain -> cli
domain -> urfave/cli
app -> urfave/cli
domain -> HTTP implementation
domain -> filesystem implementation
```

`urfave/cli` should normally only appear inside:

```text
internal/cli
```

and possibly:

```text
cmd/<app>
```

---

# 6. `main.go`

`main.go` MUST remain extremely small.

Example:

```go
package main

import (
 "context"
 "fmt"
 "os"

 appcli "example.com/project/internal/cli"
)

func main() {
 cmd := appcli.New()

 if err := cmd.Run(context.Background(), os.Args); err != nil {
  fmt.Fprintln(os.Stderr, err)
  os.Exit(1)
 }
}
```

`main.go` MUST NOT:

- contain business logic;
- parse configuration manually;
- make HTTP requests;
- contain command implementations;
- perform persistence operations.

Its job is application startup and process termination.

---

# 7. CLI Root Command

The root CLI MUST be constructed using:

```go
*cli.Command
```

Example:

```go
func New(deps Dependencies) *cli.Command {
 return &cli.Command{
  Name:    "myapp",
  Usage:   "Short description of the application",
  Version: version.Version,

  Flags: globalFlags(),

  Commands: []*cli.Command{
   newIssueCommand(deps),
   newProjectCommand(deps),
   newContextCommand(deps),
  },
 }
}
```

Prefer constructor functions rather than defining the entire CLI tree in one file.

---

# 8. Command Organization

Commands should normally be resource oriented.

Prefer:

```text
app issue list
app issue get ABC-123
app issue create
app issue delete ABC-123

app project list
app project get project-name

app context list
app context use production
```

Instead of:

```text
app list-issues
app get-issue
app create-issue
app switch-context
```

The default hierarchy is:

```text
app <resource> <action> [arguments] [flags]
```

Examples:

```text
yt issue get DEV-123
yt issue list --project DEV

tc build get 12345
tc build list --branch master

tool context use production
```

---

# 9. Commands, Arguments and Flags

All command definitions, arguments, and flags MUST use:

```text
github.com/urfave/cli/v3
```

Use typed arguments whenever possible.

Example:

```go
func newGetCommand(deps Dependencies) *cli.Command {
 return &cli.Command{
  Name:  "get",
  Usage: "Get an issue",

  Arguments: []cli.Argument{
   &cli.StringArg{
    Name: "issue",
   },
  },

  Flags: []cli.Flag{
   &cli.BoolFlag{
    Name:  "json",
    Usage: "Print result as JSON",
   },
  },

  Action: func(ctx context.Context, cmd *cli.Command) error {
   id := cmd.StringArg("issue")

   return runGet(ctx, deps, id, cmd.Bool("json"))
  },
 }
}
```

Avoid manually accessing positional arguments using:

```go
cmd.Args().Get(0)
cmd.Args().Get(1)
```

when typed `urfave/cli` arguments can express the same contract.

---

# 10. Argument vs Flag Rules

Use a positional argument when the value identifies the primary object being operated on.

Good:

```text
app issue get ABC-123
app context use production
app file inspect ./foo.json
```

Use a flag when the value modifies how the operation behaves.

Good:

```text
app issue list --project DEV
app issue get ABC-123 --json
app request send --timeout 30s
```

Prefer:

```text
app issue get ABC-123
```

over:

```text
app issue get --id ABC-123
```

unless the value is optional or one of several selectors.

---

# 11. Flag Naming

Long flags MUST use kebab-case:

```text
--dry-run
--output-file
--base-url
--page-size
```

Not:

```text
--dry_run
--outputFile
--BaseURL
```

Short aliases should only
