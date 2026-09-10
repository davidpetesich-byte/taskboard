# Taskboard

> **Fork note (branch `dp/jira-columns-labels`):** this is a local fork of
> [tcarac/taskboard](https://github.com/tcarac/taskboard) adding two board columns
> (Backlog, In Review) and a working label UI. Design and rationale live in
> `docs/superpowers/specs/2026-09-09-columns-and-labels-design.md`.

A local, self-hosted project management tool with a Kanban UI, full CLI, and a built-in [MCP](https://modelcontextprotocol.io/) server that lets AI assistants manage your projects, tickets, and teams directly.

Single binary. SQLite-backed. No Docker, no external database, no runtime dependencies.

## Screenshots

![Kanban Board](screenshots/board.png)

![Ticket Detail](screenshots/ticket-detail.png)

## Features

- **Kanban Board** — drag-and-drop ticket management across Backlog, To Do, In Progress, In Review, and Done columns
- **Labels** — colour-coded tags on tickets, settable from the ticket panel, with a board filter
- **Projects** — organize work with customizable projects (icons, colors, prefixes)
- **Teams** — assign tickets to teams
- **Tickets** — priority levels, due dates, labels, subtasks, dependencies (blocked by)
- **Embedded Terminal** — run AI coding agents (opencode, Claude Code) directly from the web UI
- **CLI** — manage everything from the terminal
- **MCP Server** — 24 tools for AI-native project management via Model Context Protocol
- **Self-Hosted** — your data stays on your machine in a SQLite database
- **Single Binary** — install this local fork with `make install`

## Install

### This local fork

This checkout is a local-only fork; upstream Homebrew v0.6.0 does not include
these five-column and label changes and must not coexist with this fork. If the
upstream binary is installed, remove it first to avoid binary shadowing and
shared database or PID-lock conflicts:

Requires Go 1.24+ and Node.js 22+.

```bash
if brew list --formula taskboard >/dev/null 2>&1; then
  brew uninstall taskboard
fi
if brew list --formula taskboard >/dev/null 2>&1; then
  echo "Homebrew taskboard is still installed; stop before installing the fork." >&2
  exit 1
fi
make install
export PATH="$HOME/.local/bin:$PATH"
hash -r
which taskboard
# expected: $HOME/.local/bin/taskboard (for example, /Users/your-user/.local/bin/taskboard)
```

## Usage

### Web UI

```bash
taskboard start
# => http://localhost:3010

taskboard start --port 8080
```

### CLI (for scripts and agents)

The data-management commands shown below accept `--json`. Commands that return data emit
API model objects; `list` commands emit arrays (`[]` when empty). Delete commands emit the
CLI confirmation object `{"deleted":true,"id":"..."}`. Errors go to stderr with a non-zero
exit code. Lifecycle and protocol commands (`start`, `stop`, `mcp`, and `clear`) are outside
this JSON contract.

Tickets may be referenced by ID or by display key (`WEB-12`); labels by ID or by an
unambiguous, case-insensitive name. If more than one label has the same name, the command
errors and requires a label ID. Due dates must use `YYYY-MM-DD`.
Optional fields that are empty (`description`, `labels`, `subtasks`, `blockedBy`, `dueDate`,
`teamId`) are omitted from the JSON rather than emitted as empty values, so treat a missing
key as empty.

| Command | Purpose |
|---|---|
| `taskboard board [--project ID]` | All five columns with their tickets |
| `taskboard ticket list [--project ID] [--team ID] [--status S] [--priority P] [--label REF]` | List tickets |
| `taskboard ticket get REF` | Full ticket: description, labels, subtasks, blockers |
| `taskboard ticket create --project ID --title T [--status S] [--priority P] [--due YYYY-MM-DD] [--team ID] [--description MD \| --description-file PATH\|-] [--label REF]...` | Create a ticket |
| `taskboard ticket update REF [--title T] [--status S] [--priority P] [--due YYYY-MM-DD] [--team ID] [--description MD \| --description-file PATH\|-] [[--label REF]... \| [--add-label REF]... [--remove-label REF]... \| --clear-labels]` | Update only the fields given |
| `taskboard ticket move REF --status S` | Change status |
| `taskboard ticket delete REF` | Delete a ticket |
| `taskboard label list` / `label create NAME [--color '#RRGGBB']` / `label delete REF` | Manage labels |
| `taskboard subtask add TICKET-REF TITLE` / `subtask toggle ID` / `subtask delete ID` | Manage subtasks |
| `taskboard project list` / `project create NAME --prefix P` / `project delete ID` | Manage projects |
| `taskboard team list` / `team create NAME` / `team delete ID` | Manage teams |

For `ticket update`, one or more repeated `--label REF` flags collectively replace the
entire label set. Repeated `--add-label REF` and `--remove-label REF` flags apply deltas to
the current set; `--clear-labels` empties it. Replace, delta, and clear modes cannot be mixed.

Statuses: `backlog`, `todo`, `in_progress`, `in_review`, `done`. Priorities: `urgent`,
`high`, `medium`, `low`. Example:

```bash
taskboard --json ticket get WEB-1 | jq .status
printf '# Plan\n\n- step one\n' | taskboard ticket update WEB-1 --description-file - --add-label Blocked
```

### MCP Server (for AI assistants)

```bash
taskboard mcp
```

#### Claude Code

```bash
claude mcp add taskboard -- /path/to/taskboard mcp
```

#### Claude Desktop

Add to `~/.claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "taskboard": {
      "command": "/path/to/taskboard",
      "args": ["mcp"]
    }
  }
}
```

#### Data Hierarchy

```
Project → Ticket → Subtask
(initiative)  (task)    (step)
```

- **Projects** are the top-level grouping (like epics/initiatives). Create one per body of work.
- **Tickets** are concrete, actionable tasks within a project. Don't create "epic" tickets — use projects.
- **Subtasks** are checklist steps within a ticket, for breaking work into verifiable pieces.

#### Available MCP Tools (24)

| Tool                    | Description                                      |
| ----------------------- | ------------------------------------------------ |
| **Projects**            |                                                  |
| `list_projects`         | List all projects with optional status filter    |
| `get_project`           | Get project details by ID                        |
| `create_project`        | Create a new project (use for epics/initiatives) |
| `update_project`        | Update project properties                        |
| `delete_project`        | Delete a project and all its tickets             |
| **Teams**               |                                                  |
| `list_teams`            | List all teams                                   |
| `get_team`              | Get team details by ID                           |
| `create_team`           | Create a new team                                |
| `update_team`           | Update team properties                           |
| `delete_team`           | Delete a team                                    |
| **Tickets**             |                                                  |
| `list_tickets`          | List tickets with filters                        |
| `get_ticket`            | Get ticket details with subtasks and labels      |
| `create_ticket`         | Create a ticket (task) within a project          |
| `update_ticket`         | Update ticket properties                         |
| `move_ticket`           | Move ticket to different status column           |
| `delete_ticket`         | Delete a ticket                                  |
| **Labels**              |                                                  |
| `list_labels`           | List all labels                                  |
| `create_label`          | Create a label (name, hex color)                 |
| `delete_label`          | Delete a label and detach it from all tickets    |
| **Board**               |                                                  |
| `get_board`             | Get full Kanban board grouped by status          |
| **Subtasks**            |                                                  |
| `create_subtask`        | Add a subtask to a ticket                        |
| `batch_create_subtasks` | Add multiple subtasks to a ticket at once        |
| `toggle_subtask`        | Toggle subtask completion                        |
| `delete_subtask`        | Remove a subtask from a ticket                   |
| `add_comment`           | Add a Markdown comment to a ticket (author defaults to `agent`) |
| `delete_comment`        | Remove a comment                                 |

#### Example Prompts

Once the MCP is connected, you can talk to your AI assistant in high-level terms and let it figure out the breakdown:

```
I'm building a SaaS billing system. Set up the project and break the work
into tickets covering Stripe integration, usage metering, invoice generation,
and a customer billing portal. Prioritize accordingly.
```

```
I need to ship a password reset flow. Think through what's involved — API
endpoints, email templates, token handling, UI screens, tests — and create
tickets with subtasks for each piece.
```

```
Look at my board and figure out what's blocking progress. If anything in
"in progress" has been sitting there without subtasks, break it down into
concrete next steps.
```

Here's what that looks like — a project and tickets created entirely by an AI assistant via MCP:

![Project created by AI](screenshots/project.png)

![Tickets list](screenshots/tickets.png)

### Embedded Terminal

The web UI includes a built-in terminal. Click **Terminal** in the sidebar to open a full PTY shell with color support and a resizable panel. Run `opencode`, `claude`, or any command directly from the browser.

![Embedded Terminal](screenshots/terminal.png)

The agent shares the same SQLite database via MCP, so tickets it creates show up on your board immediately.

## Data Storage

All data is stored in a SQLite database at:

- **macOS**: `~/Library/Application Support/taskboard/taskboard.db`
- **Linux**: `~/.config/taskboard/taskboard.db`

Migrations run automatically on first start.

### Custom Database Path

Use `--db` to point any command at a different database file:

```bash
taskboard --db /path/to/other.db start
taskboard --db /path/to/other.db mcp
taskboard --db /path/to/other.db ticket list
```

### Clearing Data

To wipe all projects, tickets, teams, and labels while keeping the schema intact:

```bash
taskboard clear        # prompts for confirmation
taskboard clear -f     # skip confirmation
```

This also respects `--db`, so you can clear a specific database file:

```bash
taskboard --db /tmp/test.db clear -f
```

## Tech Stack

| Layer        | Technology                                          |
| ------------ | --------------------------------------------------- |
| Language     | Go                                                  |
| Database     | SQLite (via modernc.org/sqlite, pure Go)            |
| CLI          | cobra                                               |
| HTTP         | chi                                                 |
| Frontend     | React, TypeScript, Tailwind CSS v4, dnd-kit         |
| Terminal     | xterm.js, gorilla/websocket, creack/pty             |
| MCP          | JSON-RPC over stdio                                 |
| Distribution | Single binary with embedded frontend via `embed.FS` |

## Development

```bash
# Run backend (serves API on :3010)
go run ./cmd/taskboard start

# Run frontend dev server (proxies API to :3010)
cd web && npm run dev

# Build everything
make build

# Clean
make clean
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

## License

[MIT](LICENSE)
