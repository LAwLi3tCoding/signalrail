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

Every segment value carries: value, confidence (`exact`, `estimated`, `cached`, `unavailable`), source, observed time, optional expiry, priority, and privacy class. Runtime adapters may omit fields but may not invent exact values.

Task state carries: title, phase, completed units, total units, status, blocker, updated time, and source runtime. Context state carries used/remaining percentage, window tokens, recent consumption samples, and optional forecast. Cost is disabled by default and always labeled estimated when sourced from a client estimate.

## Configuration contract

TOML configuration is versioned. The project file `.signalrail.toml` is shareable. Mutable handoff state lives in `.signalrail/state.json` and is ignored by default. Unknown keys fail validation with a path and suggested correction. Runtime sections may override segment order and capability policy without duplicating the full config.

## Rendering contract

The renderer must never wrap, split an ANSI sequence, leak an absolute home path in privacy mode, or emit diagnostics to protocol stdout. At constrained widths it shortens and drops lower-priority segments deterministically. Plain and ANSI outputs have equal visible width.

## Install contract

Claude installation updates a `statusLine` command entry and preserves unrelated JSON. Codex installation updates only supported `[tui]` keys using current item identifiers. Both installers create a timestamped backup, support dry-run, and report exact files changed. Project scope writes project-owned configuration only.

## Test contract

Use test-first development. Required coverage includes official-shaped Claude fixtures, malformed and partial input, width golden tests, ANSI cell widths, config precedence, task-state transitions, privacy redaction, provenance, deterministic Codex TOML generation, backup behavior, and protocol-clean stdout. CI runs format, vet, test with race detection, build, and CLI smoke tests.

## Acceptance

The implementation is accepted when all documented commands work, default output emphasizes the required fields, settings are bilingual while rendered labels stay English, 40/60/80/120/160-column tests do not wrap, Codex capability gaps are explicit, and the repository contains install documentation, examples, license, CI, and verified release-ready builds.
