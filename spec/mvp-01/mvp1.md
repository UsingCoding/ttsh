# Time Tracking TUI — MVP Specification

## 1. Goal

A fast, keyboard-driven Go application for manually tracking work time.

It provides two interfaces over the same data:

```text
TUI
    interactive daily workflow

CLI
    simple scripting / quick operations
```

Primary TUI workflow:

```text
open app
    ↓
see today's entries
    ↓
start / add / edit entries with a few keystrokes
    ↓
quit
```

Primary CLI workflow:

```text
ttsh list
ttsh add ...
ttsh view ...
ttsh remove ...
```

No database, server, daemon, account, or cloud synchronization is required.

Local state files are the source of truth and remain easy to inspect or edit manually.

---

# 2. Core concepts

## Sheet

A **sheet** represents exactly one calendar day.

Example:

```text
2026-09-29
```

Each sheet contains zero or more time entries.

On TUI startup, the current local day is selected automatically.

CLI commands also operate on the current day by default unless another date is explicitly supplied.

---

## Entry

An entry consists of:

```text
name
start
end
description
```

Example:

```text
name: YT-123
start: 09:30
end: 11:15
description: investigate Redis latency
```

Rules:

- `name` — required
- `start` — required
- `end` — optional
- `description` — optional
- time format is `HH:MM`
- an entry without `end` is considered **active/open**
- entries belong to one day only
- entries cannot span midnight

---

## Entry position / display ID

Entries do not need persistent IDs in storage for MVP.

Inside both TUI and CLI, an entry has a **display ID** corresponding to its 1-based position in the day's list:

```text
#1
#2
#3
...
```

For example:

```text
#3
```

means:

> the third entry in the currently displayed sheet.

This ID is not persisted.

If entries are reordered, inserted, or deleted, display IDs may change.

CLI commands that address an entry by ID therefore always resolve the ID against the current contents of the requested day's sheet.

---

# 3. Files

Each day gets its own file:

```text
$XDG_STATE_HOME/<app>/sheets/
├── 2026-09-27.yaml
├── 2026-09-28.yaml
└── 2026-09-29.yaml
```

Fallback:

```text
~/.local/state/<app>/sheets/
```

A file is created only after the first mutation of that day.

Recommended format:

```yaml
- name: "YT-123"
  start: "09:30"
  end: "11:15"
  description: "investigate Redis latency"

- name: "YT-456"
  start: "11:20"
  end: null
  description: ""
```

`null` means the entry is currently open.

---

# 4. Main TUI screen

Running:

```bash
ttsh
```

without a subcommand opens the TUI.

Example:

```text
┌ time ───────────────────────────────────────────── Tue 2026-09-29 ┐
│                                                                   │
│   <YT-481>   @09:00 -> 10:25   # Investigate Redis latency        │
│   <YT-517>   @10:30 -> 12:10   # Fix account service              │
│ > <Meeting>  @13:00 -> 13:45   # Architecture sync                │
│   <YT-530>   @14:05 -> ...     # Kubernetes CPU investigation     │
│                                                                   │
│                                                                   │
├───────────────────────────────────────────────────────────────────┤
│ #3 0h:45m                                    4 entries · 4h:45m   │
│ n new   e edit   c calendar   ? help   q quit                     │
└───────────────────────────────────────────────────────────────────┘
```

The entry representation is:

```text
<name> @start -> end # description
```

ASCII semantics:

```text
<YT-481>                 name
@09:00                   start
-> 10:25                 end
# investigate problem    description
```

An open entry is rendered as:

```text
<YT-530> @14:05 -> ... # Kubernetes CPU investigation
```

Lip Gloss colors/styles may additionally distinguish components, but the line must remain understandable without color.

---

# 5. Bottom status bar

The bottom status area contains:

```text
┌ selected entry ─────────────────────────────── sheet summary ┐
│ #3 0h:45m                              4 entries · 4h:45m   │
└───────────────────────────────────────────────────────────────┘
```

## Selected entry

Example:

```text
#3 0h:45m
```

means:

```text
#3       selected entry is third in the list
0h:45m   spent time for that entry
```

For a completed entry:

```text
spent = end - start
```

For an active entry:

```text
spent = current time - start
```

Example:

```text
<YT-530> @14:05 -> ...
```

At `18:20`:

```text
#4 4h:15m
```

The duration updates while the application is running.

Minute precision is enough:

```text
4h:15m
0h:25m
12h:03m
```

No seconds are displayed.

If there are no entries:

```text
--                                  0 entries · 0h:00m
```

---

## Sheet summary

The right side shows:

```text
4 entries · 4h:45m
```

The total represents completed time for the selected day.

For MVP, active entry time is not included in the total.

---

# 6. Main-screen navigation

```text
j / ↓        next entry
k / ↑        previous entry

gg           first entry
G            last entry

n            new entry
e            edit selected entry
s            stop selected active entry
c            calendar

?            help
q            quit
```

Selected entry:

```text
  <YT-481> @09:00 -> 10:25 # Redis latency
> <YT-517> @10:30 -> 12:10 # Account service
  <Meeting> @13:00 -> 13:45 # Architecture sync
```

---

# 7. Popup behavior

Popups are **not separate full-screen views**.

They are modal panels rendered over only the necessary portion of the current sheet.

The underlying sheet remains visible.

Example:

```text
┌ time ───────────────────────────────────────────── Tue 2026-09-29 ┐
│                                                                   │
│   <YT-481> @09:00 -> 10:25 # Redis latency                        │
│                                                                   │
│          ┌────────────── Edit entry ──────────────┐                │
│          │                                        │                │
│          │ Name                                   │                │
│          │   YT-517                               │                │
│          │                                        │                │
│          │ Start                                  │                │
│          │   10:30                                │                │
│          │                                        │                │
│          │ End                                    │                │
│          │   12:10                                │                │
│          │                                        │                │
│          └────────────────────────────────────────┘                │
│                                                                   │
│   <YT-530> @14:05 -> ... # Kubernetes CPU investigation           │
├───────────────────────────────────────────────────────────────────┤
│ #2 1h:40m                                    4 entries · 4h:45m   │
└───────────────────────────────────────────────────────────────────┘
```

While a popup is active:

- underlying sheet remains rendered
- sheet may be visually dimmed
- selected entry remains visible where possible
- sheet cannot receive keyboard focus
- sheet navigation is disabled
- all keyboard events go to the popup
- `Esc` closes or cancels where appropriate

This applies to:

```text
new entry
edit entry
calendar
help
confirmation dialogs
errors requiring confirmation
```

Popups should occupy only the space they need.

---

# 8. Use case 1 — create a new entry

Press:

```text
n
```

A modal opens:

```text
          ┌──────────────── New entry ────────────────┐
          │                                           │
          │ Name *                                    │
          │ > YT-                                     │
          │                                           │
          │ Suggestions                               │
          │ ┌───────────────────────────────────────┐ │
          │ │ > YT-517  Fix account service         │ │
          │ │   YT-530  Kubernetes CPU requests     │ │
          │ │   YT-551  Redis operator migration    │ │
          │ └───────────────────────────────────────┘ │
          │                                           │
          │ Start                                     │
          │   21:42                                   │
          │                                           │
          │ End                                       │
          │                                           │
          │                                           │
          │ Description                               │
          │                                           │
          ├───────────────────────────────────────────┤
          │ Tab next · Shift-Tab previous             │
          │ ↑↓ suggestions · Enter save · Esc cancel │
          └───────────────────────────────────────────┘
```

The sheet remains visible around the popup.

---

## Initial values

When `n` is pressed:

```text
name        ""
start       current HH:MM
end         empty
description ""
```

Example:

```text
start = 21:42
end   = ""
```

The user may therefore create either an active entry:

```text
Start: 21:42
End:
```

or immediately add a completed/manual entry:

```text
Start: 18:00
End:   19:30
```

This is important for cases where the user forgot to start tracking earlier and wants to enter the whole interval manually.

---

## New-entry field navigation

Fields:

```text
Name
Start
End
Description
```

Controls:

```text
Tab          next field
Shift-Tab    previous field

↑ / ↓        navigate name suggestions when applicable
Tab          may accept selected suggestion while Name is focused

Enter        submit form
Esc          cancel
```

Normal text-editing controls apply inside fields.

Description soft-wraps to the field width. When it exceeds the visible rows, the field scrolls vertically to keep the cursor visible.

---

# 9. Name suggestions

`name` remains free text.

Suggestions are optional assistance.

Typing:

```text
YT-5
```

may filter:

```text
YT-517
YT-530
YT-551
```

Users are never required to choose a suggestion.

For example:

```text
Meeting
```

remains completely valid.

---

# 10. External suggestion provider

The ttsh core has no YouTrack-specific dependency.

Conceptually:

```text
ttsh
   |
   +-- SuggestionProvider
           |
           +-- command provider
           +-- static provider
           +-- future providers
```

Possible configuration:

```toml
[suggestions]
enabled = true
command = ["some-command", "list-current-work"]
format = "lines"
timeout = "2s"
```

`lines` is the default format. Each non-empty trimmed output line is an entry name and cannot provide a description:

```text
YT-517
YT-530
YT-551
```

Use `format = "json"` when a suggestion needs explanatory text. The command must emit one top-level JSON array of objects; each object has a required `name` and an optional `description`:

```json
[
  {
    "name": "YT-517",
    "description": "Fix account service"
  }
]
```

The form displays the description beside the suggestion name. Accepting a suggestion fills only the Name field; its description is not copied into the entry description or persisted.

Provider failure must never prevent manual entry.

---

# 11. New-entry validation

On `Enter`:

```text
name != empty

start:
    valid HH:MM

end:
    empty
    OR valid HH:MM
```

If `end` is present:

```text
end >= start
```

Examples:

Valid active entry:

```text
Start: 14:10
End:
```

Valid completed entry:

```text
Start: 14:10
End:   15:45
```

Invalid:

```text
Start: 14:10
End:   13:45
```

On successful save:

```text
save
    ↓
close popup
    ↓
refresh sheet
    ↓
select newly created entry
```

---

# 12. Use case 2 — edit an entry

Select:

```text
  <YT-481> @09:00 -> 10:25 # Redis latency
> <YT-517> @10:30 -> 12:10 # Fix account service
```

Press:

```text
e
```

Popup:

```text
          ┌─────────────── Edit entry ────────────────┐
          │                                           │
          │ Name                                      │
          │   YT-517                                  │
          │                                           │
          │ Start                                     │
          │   10:30                                   │
          │                                           │
          │ End                                       │
          │   12:10                                   │
          │                                           │
          │ Description                               │
          │   Fix account service                     │
          │                                           │
          ├───────────────────────────────────────────┤
          │ Tab next · Shift-Tab previous             │
          │ Enter save · Esc cancel                   │
          └───────────────────────────────────────────┘
```

Everything is editable.

Validation is the same as for New Entry.

Overlapping entries remain allowed for MVP.

---

# 13. Open/running entry

An entry without an end:

```text
<YT-530> @14:05 -> ... # CPU investigation
```

Storage:

```yaml
- name: "YT-530"
  start: "14:05"
  end: null
  description: "CPU investigation"
```

At `18:20`, if selected:

```text
#4 4h:15m
```

The displayed duration updates at least once per minute.

---

# 14. Stop active entry

Press:

```text
s
```

on the selected active entry.

Before:

```text
<YT-530> @14:05 -> ...
```

At `16:45` after `s`:

```text
<YT-530> @14:05 -> 16:45
```

Footer:

```text
#4 2h:40m
```

---

# 15. Starting an entry while another is active

Only one active entry should exist by default.

If an active entry exists and the user creates another entry **without an End time**, confirmation appears:

```text
          ┌────────────────────────────────────┐
          │ Another entry is currently active │
          │                                    │
          │ <YT-530> started at 14:05          │
          │                                    │
          │ Stop it at 16:22 and start new?    │
          │                                    │
          │        y yes      n cancel          │
          └────────────────────────────────────┘
```

A newly created entry that already has an `End` value does not become active and therefore does not require stopping the existing entry.

Future configuration:

```toml
allow_parallel_entries = false
```

---

# 16. Use case 3 — calendar

Press:

```text
c
```

Calendar popup:

```text
        ┌───────────── September 2026 ─────────────┐
        │                                          │
        │ Mon  Tue  Wed  Thu  Fri  Sat  Sun        │
        │                                          │
        │      01   02   03   04   05   06         │
        │ 07   08   09   10   11   12   13         │
        │ 14   15   16   17   18   19   20         │
        │ 21   22   23   24   25   26   27         │
        │ 28  [29*] 30                              │
        │                                          │
        │ * has entries                            │
        ├──────────────────────────────────────────┤
        │ hjkl move · PgUp/PgDn month · t today   │
        │ Enter open · Esc cancel                  │
        └──────────────────────────────────────────┘
```

`*` indicates a day containing entries.

Navigation:

```text
h / ←        previous day
l / →        next day
k / ↑        previous week
j / ↓        next week

PgUp         previous month
PgDn         next month

t            today

Enter        open selected day
Esc          cancel
```

On `Enter`:

```text
release current sheet lock
        ↓
acquire selected sheet lock
        ↓
load selected sheet
        ↓
close calendar
```

---

# 17. Empty day

```text
┌ time ─────────────────────────────────────────── Sun 2026-09-27 ┐
│                                                                 │
│                                                                 │
│                       No entries yet                            │
│                                                                 │
│                        n new entry                              │
│                        c calendar                               │
│                                                                 │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ --                                           0 entries · 0h:00m │
│ n new   c calendar   ? help   q quit                            │
└─────────────────────────────────────────────────────────────────┘
```

---

# 18. CLI interface

The non-interactive CLI covers simple operations without opening Bubble Tea.

Top-level shape:

```text
ttsh
ttsh list
ttsh view
ttsh add
ttsh remove
```

Running only:

```bash
ttsh
```

opens the TUI.

CLI commands operate on today's sheet by default.

A common date option is available:

```bash
--date YYYY-MM-DD
```

Example:

```bash
ttsh list --date 2026-09-25
```

Every non-interactive command accepts a per-command `--json` flag. Successful `list --json` responses are a JSON array; successful `view`, `add`, and `remove` responses are one JSON entry object. Every successful response is newline-terminated.

Each entry always has `id` (display ID), `date` (`YYYY-MM-DD`), `name`, `start` (`HH:MM`), `end` (`HH:MM` or `null` for an active entry), `description` (including `""`), and `duration` (for example, `0h:45m`). An empty JSON list is `[]`. JSON mode does not wrap responses or change failures: errors remain on stderr with a non-zero exit status.

---

# 19. CLI — list entries

Basic command:

```bash
ttsh list
```

Example output:

```text
#1 <YT-481>  @09:00 -> 10:25  # Investigate Redis latency
#2 <YT-517>  @10:30 -> 12:10  # Fix account service
#3 <Meeting> @13:00 -> 13:45  # Architecture sync
#4 <YT-530>  @14:05 -> ...    # Kubernetes CPU investigation
```

Optional footer:

```text
4 entries · 4h:45m
```

Specific day:

```bash
ttsh list --date 2026-09-28
```

An empty day:

```text
No entries for 2026-09-28.
```

`list` is intended for quickly inspecting the whole day.

---

# 20. CLI — view an entry

View one entry by display ID:

```bash
ttsh view 3
```

Equivalent explicit form:

```bash
ttsh view 3 --date 2026-09-29
```

Example output:

```text
ID:          #3
Name:        Meeting
Start:       13:00
End:         13:45
Spent:       0h:45m
Description: Architecture sync
```

For an active entry:

```text
ID:          #4
Name:        YT-530
Start:       14:05
End:         -
Spent:       4h:15m
Description: Kubernetes CPU investigation
```

A convenient `#` prefix may also be accepted:

```bash
ttsh view '#3'
```

but internally:

```text
3
#3
```

mean the same display position.

Invalid position:

```text
entry #9 does not exist for 2026-09-29
```

with a non-zero exit code.

---

# 21. CLI — add an entry

Minimal case:

```bash
ttsh add YT-530
```

Equivalent semantics:

```text
name  = YT-530
start = current HH:MM
end   = empty
desc  = empty
```

Example output:

```text
Added #5 <YT-530> @21:42 -> ...
```

---

## Add with description

```bash
ttsh add YT-530 --description "Investigate CPU load"
```

Short option may later be:

```bash
-d
```

---

## Add with explicit start

```bash
ttsh add YT-530 --start 14:10
```

---

## Add completed/manual entry

```bash
ttsh add YT-530 \
  --start 14:10 \
  --end 15:45 \
  --description "Investigate CPU load"
```

Result:

```text
Added #5 <YT-530> @14:10 -> 15:45 # Investigate CPU load
```

---

## Add to another day

```bash
ttsh add YT-530 \
  --date 2026-09-28 \
  --start 14:10 \
  --end 15:45
```

---

## Add command options

MVP:

```text
ttsh add <name>

--date <YYYY-MM-DD>
--start <HH:MM>
--end <HH:MM>
--description <text>
```

Defaults:

```text
date        today
start       current local HH:MM
end         empty
description empty
```

The same domain validation used by the TUI must apply.

---

## Existing active entry

If another active entry exists and:

```bash
ttsh add ...
```

would create another active entry, the non-interactive CLI must not silently alter existing data.

For MVP, fail explicitly:

```text
cannot start entry: #4 <YT-530> is already active
```

with a non-zero exit code.

The user can:

- stop/edit the previous entry in the TUI
- or add a completed entry with `--end`

The CLI should avoid interactive confirmation prompts by default.

---

# 22. CLI — remove an entry

Remove by display ID:

```bash
ttsh remove 3
```

Specific date:

```bash
ttsh remove 3 --date 2026-09-28
```

Example output:

```text
Removed #3 <Meeting> @13:00 -> 13:45
```

No confirmation is required for the basic MVP command because it is intended to be scriptable and explicit.

Invalid ID:

```text
entry #7 does not exist for 2026-09-29
```

with non-zero exit code.

After deletion, following display IDs shift naturally.

Example before:

```text
#1 A
#2 B
#3 C
#4 D
```

After:

```bash
ttsh remove 2
```

the list becomes:

```text
#1 A
#2 C
#3 D
```

---

# 23. CLI behavior summary

Examples:

```bash
# show today
ttsh list

# show another day
ttsh list --date 2026-09-28

# inspect entry
ttsh view 3

# start work now
ttsh add YT-530

# start with description
ttsh add YT-530 --description "Redis investigation"

# manually add finished work
ttsh add Meeting \
  --start 13:00 \
  --end 13:45 \
  --description "Architecture sync"

# add something to yesterday manually
ttsh add YT-481 \
  --date 2026-09-28 \
  --start 09:00 \
  --end 10:25

# remove mistaken entry
ttsh remove 3
```

These commands intentionally cover straightforward scenarios only.

Complex editing remains primarily a TUI workflow in MVP.

---

# 24. Shared CLI/TUI application layer

CLI and TUI must not implement business rules independently.

Conceptually:

```text
                     ┌────────────────┐
                     │ application    │
                     │ services       │
                     └───────┬────────┘
                             │
              ┌──────────────┴──────────────┐
              │                             │
              v                             v
          TUI adapter                   CLI adapter
        Bubble Tea UI                  Cobra commands
```

Shared operations should conceptually resemble:

```text
ListEntries(date)
GetEntry(date, index)
AddEntry(date, input)
UpdateEntry(date, index, input)
RemoveEntry(date, index)
StopEntry(date, index, now)
```

This guarantees identical:

- validation
- locking
- YAML persistence
- active-entry rules
- duration calculations

across both interfaces.

---

# 25. Per-day locking

Only one process may manage a sheet at a time.

Files:

```text
$XDG_STATE_HOME/<app>/sheets/
├── 2026-09-29.yaml
└── 2026-09-29.lock
```

The `.lock` file uses an OS advisory exclusive lock.

Conceptually:

```text
select date
    ↓
open/create <date>.lock
    ↓
acquire exclusive lock
    ↓
perform operation
```

---

## TUI lock lifetime

The TUI keeps the lock for as long as the day is active.

```text
open day
    ↓
acquire lock
    ↓
work with sheet
    ↓
switch day / quit
    ↓
release lock
```

---

## CLI lock lifetime

CLI commands hold the lock only for the duration of the operation.

Example:

```text
ttsh add ...
    ↓
acquire day lock
    ↓
read
    ↓
mutate
    ↓
atomic write
    ↓
release lock
    ↓
exit
```

Even read-oriented commands such as `view` and `list` should use the same locking abstraction for MVP simplicity and deterministic behavior.

Therefore, if that day's sheet is currently managed by a running TUI instance:

```bash
ttsh add YT-123
```

fails rather than racing with it.

Example:

```text
2026-09-29 is currently locked by another ttsh process
```

---

# 26. Lock file contents

Lock file may contain informational metadata:

```yaml
pid: 18421
hostname: VASILIYUSPEX
started_at: 2026-09-29T21:42:00+03:00
```

Actual ownership is determined by the OS advisory lock.

The presence of `.lock` alone does not indicate that a sheet is actively locked.

---

# 27. Persistence behavior

Every mutation is immediately persisted.

Examples:

```text
TUI create
TUI edit
TUI stop

CLI add
CLI remove
```

No explicit Save command is necessary.

---

## Atomic writes

```text
serialize sheet
      ↓
write temporary file
      ↓
flush / close
      ↓
rename over original
```

For example:

```text
2026-09-29.yaml.tmp
        ↓
rename
        ↓
2026-09-29.yaml
```

Locks and atomic writes solve different problems:

```text
.lock
    prevents concurrent ttsh processes

atomic rename
    prevents partial/corrupted files
```

Both are required.

---

# 28. Manual external editing

Files remain human-editable.

However, while a ttsh process owns the day's lock, manual editing of that day's YAML is outside the supported concurrent-write model.

The user should close or switch away from that sheet before editing the YAML manually.

This avoids merge/reload complexity in MVP.

---

# 29. Configuration

Configuration:

```text
$XDG_CONFIG_HOME/<app>/config.toml
```

Fallback:

```text
~/.config/<app>/config.toml
```

Example:

```toml
allow_parallel_entries = false

[suggestions]
enabled = true
command = ["some-command", "list-current-work"]
format = "lines"
timeout = "2s"
```

State and configuration remain separate.

---

# 30. Help popup

Press:

```text
?
```

Popup:

```text
          ┌──────────────── Help ────────────────┐
          │                                     │
          │ Navigation                          │
          │   j / ↓       next entry            │
          │   k / ↑       previous entry        │
          │   gg          first entry           │
          │   G           last entry            │
          │                                     │
          │ Entries                             │
          │   n           new                   │
          │   e           edit                  │
          │   s           stop active entry     │
          │                                     │
          │ Sheets                              │
          │   c           calendar              │
          │                                     │
          │ General                             │
          │   ?           help                  │
          │   q           quit                  │
          │                                     │
          │                 Esc close           │
          └─────────────────────────────────────┘
```

The sheet remains visible behind it.

---

# 31. Tech stack

Recommended:

```text
Cobra
    CLI commands / flags

Bubble Tea
    TUI application state / update loop

Bubbles
    textinput
    list
    spinner
    help
    viewport

Lip Gloss
    popup borders
    focus state
    text emphasis
    colors
    layout
```

Forms can initially use `bubbles/textinput` directly.

This gives precise control over:

```text
Enter
Esc
Tab
suggestions
modal focus
```

---

# 32. Application modes

Conceptually:

```go
type Mode int

const (
    ModeSheet Mode = iota
    ModeNew
    ModeEdit
    ModeCalendar
    ModeHelp
    ModeConfirm
)
```

`ModeNew`, `ModeEdit`, etc. are not separate screens.

They represent:

```text
base sheet
    +
active popup
```

Conceptual TUI state:

```text
App
├── selectedDate
├── entries
├── selectedEntry
├── sheetLock
├── mode
│
├── sheet
├── entryForm
├── calendar
├── suggestions
├── confirmation
└── status
```

Rendering:

```text
View()
  |
  +-- render Sheet
  |
  +-- if mode != ModeSheet
          render Popup over Sheet
```

Input:

```text
ModeSheet
    ↓
sheet receives keys

ModeNew/Edit/Calendar/...
    ↓
popup receives keys
    ↓
sheet receives none
```

---

# 33. Suggested package boundaries

```text
cmd/
    root.go
    list.go
    view.go
    add.go
    remove.go

internal/
    app/
        service.go
        entry.go

    tui/
        model.go
        update.go
        view.go
        keys.go

    sheet/
        entry.go
        sheet.go
        validation.go
        timeofday.go

    storage/
        storage.go
        yaml.go
        paths.go
        lock.go

    ui/
        sheet/
        entryform/
        calendar/
        help/
        confirm/
        popup/

    suggestions/
        provider.go
        command.go

    config/
        config.go
```

Important separation:

```text
cmd/
    CLI presentation only

tui/
    Bubble Tea presentation only

app/
    operations/use cases

sheet/
    domain

storage/
    persistence + locking
```

---

# 34. Domain model

Conceptually:

```go
type Entry struct {
    Name        string
    Start       TimeOfDay
    End         *TimeOfDay
    Description string
}

type Sheet struct {
    Date    time.Time
    Entries []Entry
}
```

`#3` is not part of `Entry`.

It is calculated:

```go
displayID := index + 1
```

A `TimeOfDay` value type handles:

```text
09:30
21:05
```

instead of spreading arbitrary strings through the application.

---

# 35. Normal daily workflow

TUI:

```bash
ttsh
```

```text
  <TASK-1> @09:00 -> 10:30 # implementation
> <TASK-2> @10:45 -> ...
```

Footer:

```text
#2 1h:20m                         2 entries · 1h:30m
```

Start something:

```text
n
type or select name
optional end
optional description
Enter
```

Correct something:

```text
j/k
e
edit
Enter
```

Stop:

```text
s
```

Change day:

```text
c
hjkl
Enter
```

CLI quick operations:

```bash
ttsh list
ttsh view 2
ttsh add TASK-3
ttsh remove 2
```

---

# 36. MVP scope

## Required

```text
[x] current-day TUI sheet on startup
[x] one YAML file per day
[x] one exclusive .lock per active operation/day
[x] list entries
[x] selected-entry display ID (#N)
[x] selected-entry live spent time
[x] daily completed-time total
[x] j/k navigation
[x] create entry
[x] New Entry supports Start and optional End
[x] name suggestions
[x] external suggestion command
[x] edit entry
[x] stop active entry
[x] calendar
[x] arbitrary historical/future sheets
[x] partial-screen modal popups
[x] underlying sheet remains visible during popup
[x] XDG state/config directories
[x] atomic writes
[x] help popup

[x] CLI list
[x] CLI view
[x] CLI add
[x] CLI remove
[x] common --date support
[x] shared domain/application logic between CLI and TUI
```

## Outside MVP

```text
[ ] Google Sheets synchronization
[ ] YouTrack built directly into ttsh
[ ] reports
[ ] weekly/monthly statistics
[ ] tags
[ ] projects
[ ] billing
[ ] daemon/background timer
[ ] cloud sync
[ ] read-only second process
[ ] multiple simultaneous timers by default
[ ] cross-midnight entries
[ ] merging concurrent manual YAML changes
[ ] complex CLI editing workflow
```

---

# 37. Overall architecture

```text
                      ┌──────────────────┐
                      │      Domain      │
                      │ Sheet / Entry    │
                      │ TimeOfDay        │
                      └────────┬─────────┘
                               │
                      ┌────────▼─────────┐
                      │ Application      │
                      │ Services         │
                      │                 │
                      │ List / Get       │
                      │ Add / Update     │
                      │ Remove / Stop    │
                      └──────┬───┬──────┘
                             │   │
                  ┌──────────┘   └──────────┐
                  │                         │
          ┌───────▼────────┐       ┌────────▼───────┐
          │      TUI       │       │      CLI       │
          │ Bubble Tea     │       │     Cobra      │
          │ Bubbles        │       │                │
          │ Lip Gloss      │       │ list/view/add  │
          └────────────────┘       │ remove         │
                                   └────────────────┘
                             │
                             ▼
                  ┌─────────────────────┐
                  │ Storage             │
                  │ YAML + XDG paths    │
                  │ .lock + atomic save │
                  └─────────────────────┘
```

The central UI rule remains:

```text
Sheet is always the TUI base view.

Popups temporarily take input focus,
but do not replace the sheet visually.
```

And the central architecture rule is:

```text
TUI and CLI are only adapters.

All time-entry behavior lives in one
shared application/domain implementation.
```

---

# 38. MVP-1 Addendum — TUI Layout and Visual Style

This addendum refines the interactive presentation only. Existing interaction,
CLI, persistence, locking, application, and domain rules remain authoritative.

## 38.1. Purpose

The TUI is a full alternate-screen daily sheet with compact information density.
Its application identity is `ttsh`.

## 38.2. Alternate-screen behavior

`ttsh` enters the terminal alternate screen when the interactive adapter starts
and restores the preceding shell contents cleanly when it exits, including
after `q`, cancellation, or an error. Terminal control remains owned by the
Bubble Tea program launch path.

## 38.3. Viewport contract

`tea.WindowSizeMsg` is the only source of viewport dimensions. Once a positive
width and height are received, the rendered view occupies exactly that many
terminal cells. A resize immediately redraws the full canvas and reflows an
open form's fields. Before dimensions arrive, the view remains a compact
intrinsic representation rather than a fabricated zero-sized frame.

## 38.4. Compact density

At approximately 80×24, retain one compact header row, a scrollable entry
body, and two footer rows. Keep the entry region left aligned and no wider than
96 columns; wider terminals add breathing room rather than stretched names,
descriptions, or form fields.

## 38.5. Visual direction

Use one built-in dark muted-slate presentation: a dark blue-slate canvas, soft
off-white foreground, desaturated slate metadata and borders, restrained
cyan-blue accent, subtly lighter selected and status surfaces, muted
green/cyan running state, and restrained red errors. This is a visual
direction, not a copy of another application's branding, palette, glyphs,
layout, or source.

## 38.6. Semantic theme

TUI palette literals belong to one built-in semantic theme. Its roles are
`Background`, `Panel`, `Foreground`, `Muted`, `Border`, `Accent`, `Cursor`,
`Selected`, `StatusBar`, `StatusText`, `EntryName`, `EntryStart`, `EntryEnd`,
`EntryDescription`, `Running`, and `Error`. Renderers consume reusable styles
derived from those roles; component-local color construction is not allowed.

## 38.7. Header and footer

The header shows a minimal `ttsh` identity and selected date with sparing
accent. The first footer row shows the selected entry's `#N <live duration>`
at left and `<count> entries · <completed total>` at right when both fit. The
second footer row retains exactly the `n`, `e`, `s`, `c`, `?`, and `q` hints,
with accent keys and muted labels. At narrow widths labels may be shortened,
but their keys remain; selected-entry status takes precedence over the sheet
summary.

## 38.8. Sheet row layout

Every entry has an explicit non-color-readable selected marker (`>`), name,
`start → end` interval, and optional description. The selected row combines
the marker, stronger foreground, and selected surface. An open entry still
uses `...` for its end and a distinct running style; it does not animate.

## 38.9. Empty sheet

An empty sheet displays `No entries today` and `n create entry` within the
same full canvas. It is not an unfinished blank screen.

## 38.10. Responsive clipping and scrolling

Text is clipped to its allocated display width with an ellipsis. The visible
entry rows are one contiguous window that keeps the selected zero-based entry
visible and indicates omitted rows above and below without changing selection
or session data.

## 38.11. Popup composition

New/Edit, Calendar, Help, active-entry confirmation, and Error views are
compact bordered panel overlays centered and clamped in the viewport. They are
composed over the existing sheet canvas, preserving every uncovered base cell;
modal text is never appended below the sheet. Preferred widths are 60 for
form/edit, 40 for calendar, 64 for help, and 60 for confirmation/error, capped
at viewport width minus four cells. Popup height is content height capped at
viewport height minus four cells.

## 38.12. Form presentation

Forms use labels above unboxed fields, accent focus, muted inactive labels,
and bounded suggestions with an explicit selected marker. Bubbles text inputs
use the shared theme for prompt, cursor, text, and related field styling.
Existing Tab, Shift-Tab, Enter, Esc, suggestion, validation, cancellation, and
save behavior remains unchanged.

## 38.13. Calendar presentation

The calendar is a fixed-width Monday-first grid. Its five-cell day slots retain
brackets for selection and `*` for dates with entries. Selection brackets take
precedence. Selected, today, marked, and normal dates have separate semantic
styles. Existing calendar navigation, opening, cancellation, day switching,
and session recovery remain unchanged.

## 38.14. Small popup policy

When a terminal is smaller than a popup's preferred height, remove optional
suggestion rows and secondary hints before removing the title, focused content,
form errors, or close/save instruction. Popup bounds must never exceed the
canvas.

## 38.15. Non-color compatibility

Color supplements, never carries, meaning. Selection, active state, entry
position, and time interval remain readable through text markers, labels,
brackets, `*`, and `...`.

## 38.16. Scope exclusions

This addendum does not change time-tracking behavior, CLI output or commands,
YAML format, persistence, path resolution, locks, session ownership, domain
validation, suggestion-provider behavior, or the existing modal state machine.
It adds no animation, custom terminal escape handling, second UI/rendering
library, new theme selector, copied external branding, or unrelated product
features.

## 38.17. Acceptance environment

Acceptance includes a real terminal at 80×24 with isolated XDG state, then a
larger resize and a return to 80×24. Add one completed and one active entry
through the existing CLI commands before checking populated-sheet behavior.

## 38.18. Acceptance criteria 1–7

1. TUI launch enters the alternate buffer and exit restores the shell surface.
2. A received 80×24 `WindowSizeMsg` renders an 80×24 canvas.
3. A larger received viewport renders to its exact dimensions and returns to
   80×24 without stale content.
4. The 80×24 view retains header, scrollable body, and both footer rows.
5. Wide terminals preserve compact left-aligned entry density.
6. Empty sheets show the styled empty-state hint and creation instruction.
7. Populated sheets show distinct name, time, description, selected, and
   running treatment without relying on color.

## 38.19. Acceptance criteria 8–14

8. A selected active row visibly retains `>`, `...`, and its `#N` live-status
   marker.
9. Footer semantics retain `--`, zero-entry behavior, and `0h:00m`.
10. Long text is clipped rather than escaping the allocated viewport region.
11. A long list keeps the selected entry visible and signals omitted rows.
12. New/Edit, Calendar, Help, confirmation, and Error panels remain within the
    canvas and leave visible sheet content outside their borders.
13. Form suggestions, focused fields, errors, calendar brackets, and marked
    dates retain explicit non-color markers and existing key behavior.
14. Resize, cancellation, save, stop, calendar day switch, error recovery,
    `q`, and clean terminal restoration preserve all existing behavior.
