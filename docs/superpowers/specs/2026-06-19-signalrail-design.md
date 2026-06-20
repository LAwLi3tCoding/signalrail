# SignalRail Product and Technical Specification

## Decision

Build SignalRail as a Go 1.23+ single-binary CLI. The selected UI is the one-line Signal Rail described in `DESIGN.md`. Claude Code receives the full renderer through its official command status-line contract. Codex receives a generated native status-line configuration because current Codex releases do not expose an external command renderer.

## Alternatives considered

1. TypeScript/Node: strongest JSON/TUI ecosystem and fastest iteration, but runtime installation and cold-start overhead weaken a frequently invoked status command.
2. Rust: best raw performance and strong terminal libraries, but higher implementation and contribution cost for the settings and adapter surface.
3. Go: selected compromise—single static binary, low startup latency, simple cross-compilation, strong testing, and sufficient TUI support.

## Required commands

- `signalrail render [--runtime auto|claude|generic] [--width N] [--format ansi|plain|json]`
- `signalrail preview [--preset wide|standard|compact|minimal]`
- `signalrail config [--lang en|zh-CN]`
- `signalrail task set|step|block|done|clear|show`
- `signalrail install claude|codex [--scope user|project] [--dry-run]`
- `signalrail explain [--json]`
- `signalrail doctor [--json]`

## Normalized data model

Every segment value carries: value, confidence (`exact`, `estimated`, `unavailable`), freshness (`fresh`, `cached`, `stale`, `degraded`), source, observed time, optional expiry, priority, and privacy class. Runtime adapters may omit fields but may not invent exact values. `↻` means cached and within expiry; `!` means stale or degraded.

Task state carries: title, phase, completed units, total units, status, blocker, updated time, and source runtime. Context state carries used/remaining percentage, window tokens, recent consumption samples, and optional forecast. Cost is disabled by default and always labeled estimated when sourced from a client estimate.

## Configuration contract

TOML configuration is versioned. The project file `.signalrail.toml` is shareable. Mutable handoff state lives in `.signalrail/state.json` and is ignored by default. Unknown keys fail validation with a path and suggested correction. Runtime sections may override segment order and capability policy without duplicating the full config.

## Rendering contract

The renderer must never wrap, split an ANSI sequence, leak an absolute home path in privacy mode, or emit diagnostics to protocol stdout. At constrained widths it shortens and drops lower-priority segments deterministically. Plain and ANSI outputs have equal visible width.

## Install contract

Claude installation updates a `statusLine` command entry and preserves unrelated JSON. Codex installation targets the item set pinned in `DESIGN.md` at upstream commit `406062c3af8b27c8e1b4b83c485ebe1ae0df874c`; native `task-progress` comes from Codex `update_plan` and cannot read SignalRail task state. Both installers create a timestamped backup, support dry-run, and report exact files changed.

## Test contract

Use test-first development. Required coverage includes official-shaped Claude fixtures, malformed and partial input, width golden tests, ANSI cell widths, config precedence, task-state transitions, privacy redaction, provenance, deterministic Codex TOML generation, backup behavior, and protocol-clean stdout. CI runs format, vet, test with race detection, build, and CLI smoke tests.

## Acceptance

- `render`: exit 0 and exactly one stdout line; malformed/oversized input exits 2 with stderr only.
- `preview`: exit 0 without stdin; unknown preset exits 2 and writes no files.
- `config`: English and Chinese menus persist valid TOML; cancel exits 0 without writes; invalid language exits 2.
- `task`: valid transitions atomically update `.signalrail/state.json`; invalid transitions exit 2; `clear` removes only that state file.
- `install`: dry-run never writes; apply preserves unrelated keys and creates a backup; malformed config exits 2 without writes; strict unsupported Codex capability exits 3.
- `explain`: text and JSON list source, confidence, freshness, inclusion decision, and age for every candidate segment.
- `doctor`: exit 0 when healthy, 1 for actionable warnings, and 2 for unusable configuration; JSON remains machine-readable.
- Default output emphasizes required fields, settings are bilingual while rendered labels stay English, and 40/60/80/120/160-cell tests never wrap.
- Privacy and provenance tests, performance benchmarks, Codex capability warnings, documentation, examples, license, CI, and source builds satisfy `DESIGN.md`.

## v1 cutline

v1 is the command set above with one-line rendering and local data only. Multi-line UI, live custom Codex rendering, network usage APIs, daemons/watchers, IDE surfaces, signed release artifacts, Homebrew/Scoop/npm publication, and web UI are explicitly deferred.
