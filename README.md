# SignalRail

SignalRail is a trustworthy status line for AI coding sessions. It keeps model, project, task, progress, and context pressure in the first scan path, while making estimates and stale data explicit.

```text
◆ GPT-5.5 · high  ◇ signalrail/main  ▶ Build renderer  3/7 ███░░░░  CTX 38% left  $1.24~
```

## Why SignalRail

- One semantic vocabulary for Claude Code and Codex.
- Width-safe output at 40, 60, 80, 120, and 160 terminal cells.
- Exact, estimated, cached, stale, and degraded values are visibly distinct.
- Project task state supports handoff between sessions and tools.
- English status line; English and Simplified Chinese settings.
- No render-time network access, telemetry, account identity, or arbitrary segment shell commands.

## Runtime support

| Runtime | Integration | Capability |
|---|---|---|
| Claude Code | Official command `statusLine` JSON stdin | Full SignalRail renderer, ANSI, task state, context, optional cost |
| Codex CLI | Native `[tui].status_line` compiler | Native model/project/branch/task-progress/context/limit fields and colors |

Codex does not currently expose a command-backed custom status line. SignalRail does not pretend otherwise: `install codex` generates supported native fields, while `preview`, `task show`, and `explain` remain available from any shell. Codex native `task-progress` reflects Codex `update_plan`, not SignalRail's project state.

## Install

Requires Go 1.23 or newer:

```bash
go install github.com/LAwLi3t-CN/signalrail/cmd/signalrail@latest
```

Or build from source:

```bash
git clone https://github.com/LAwLi3t-CN/signalrail.git
cd signalrail
go build -o bin/signalrail ./cmd/signalrail
```

## Quick start

```bash
# Interactive English settings
signalrail config --lang en --scope project

# 中文设置
signalrail config --lang zh-CN --scope project

# Preview the selected UI without changing runtime configuration
signalrail preview --preset standard

# Install integrations; inspect first with --dry-run
signalrail install claude --scope user --dry-run
signalrail install claude --scope user
signalrail install codex --scope user --dry-run
signalrail install codex --scope user
```

Installers preserve unrelated settings, reject symlinks and stale plans, create timestamped backups, and write atomically.

## Task handoff

```bash
signalrail task set "Build renderer" --phase coding --total 7
signalrail task step
signalrail task block "Waiting for review"
signalrail task show
signalrail task done
signalrail task clear
```

Mutable state is stored in `.signalrail/state.json`; the repository `.gitignore` excludes it by default. Shared policy belongs in `.signalrail.toml`.

## Commands

| Command | Purpose |
|---|---|
| `render` | Read Claude or normalized JSON and emit ANSI, plain, or JSON output |
| `preview` | Render wide, standard, compact, or minimal sample data |
| `config` | Keyboard-only bilingual settings menu with preview and confirmation |
| `task` | Set, step, block, complete, clear, or inspect project task state |
| `install` | Safely install Claude Code or compile Codex native configuration |
| `explain` | Show inclusion, source, confidence, freshness, and age for every segment |
| `doctor` | Validate SignalRail policy and effective runtime integrations |

Exit codes are stable: `0` success, `1` actionable doctor warning, `2` invalid input/configuration, and `3` strict Codex capability mismatch.

## Configuration

Precedence is built-in defaults, user config, project config, runtime profile, then CLI flags.

- User: `$XDG_CONFIG_HOME/signalrail/config.toml`, falling back to `~/.config/signalrail/config.toml`
- Project: `.signalrail.toml`

Arrays replace lower layers; tables merge. Unknown keys fail validation. See [`examples/standard.toml`](examples/standard.toml) and [`examples/privacy-first.toml`](examples/privacy-first.toml).

## Provenance markers

| Marker | Meaning |
|---|---|
| none | exact and fresh |
| `~` | estimated |
| `↻` | cached within TTL |
| `!` | stale or degraded |

Use `signalrail explain --json` for the source and age of every segment. Cost is disabled by default and is labeled estimated when supplied by a client.

## Privacy

Privacy mode is on by default. It redacts home paths and usernames and supports sensitive branch patterns. SignalRail performs no render-time network requests and never displays account email, session identity, or organization by default.

## Compatibility

- Primary: macOS and Linux.
- CI: macOS, Linux, and Windows, including Bash or PowerShell CLI smoke checks.
- Unicode and ANSI are optional; `--ascii`, `--no-color`, and `NO_COLOR` are supported.
- No Nerd Font is required.
- Codex item compatibility is pinned in [`DESIGN.md`](DESIGN.md) to the documented upstream commit.

## Troubleshooting

```bash
signalrail doctor
signalrail doctor --json
signalrail explain
signalrail install claude --dry-run
signalrail install codex --dry-run
```

If Claude Code does not display the line, confirm workspace trust is accepted and `disableAllHooks` is not enabled. If Codex omits a field, the field may be unavailable or unsupported by its native status line; SignalRail reports this instead of inventing a value.

## Development

```bash
gofmt -w .
go vet ./...
go test -race ./...
go test -bench Render -benchmem ./internal/render
go build ./cmd/signalrail
./scripts/smoke.sh
```

Architecture, alternatives, UI choices, competitor research, and the implementation contract are documented in [`DESIGN.md`](DESIGN.md) and [`docs/`](docs/).

## License

MIT
