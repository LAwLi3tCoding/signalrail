# SignalRail Design

## Source of truth

- Status: Active
- Last refreshed: 2026-06-19
- Primary surfaces: Claude Code status line, Codex native status configuration, standalone CLI preview and diagnostics
- Evidence reviewed: official Claude Code status-line documentation, current OpenAI Codex source/config schema and issues, and the competitor matrix in `docs/research/competitive-landscape.md`

## Brand

- Personality: calm, precise, operational, trustworthy
- Trust signals: metric provenance, freshness markers, explicit capability gaps, deterministic rendering
- Avoid: identity-first content, decorative dashboards, fake precision, mandatory Nerd Fonts, hidden network calls

## Product goals

- Put model, project, active task, progress, and context pressure in the first scan path.
- Provide one semantic vocabulary across Claude Code and Codex while exposing runtime-specific data and capability limits.
- Remain readable at 80, 120, and 160 columns without wrapping.
- Make estimated, cached, stale, and unavailable values visibly different from exact values.
- Keep cost optional and identity information off by default.

## Non-goals

- Pretending Codex supports an external command-backed status line before upstream provides one.
- Replacing tmux, zellij, Starship, or a full observability platform.
- Polling billing APIs or uploading telemetry by default.
- Supporting arbitrary user shell execution inside render segments in v1.

## Personas and jobs

- Primary: developers running long agentic coding sessions in terminal IDEs.
- Secondary: teams that want a shared project status-line policy without sharing user identity or local paths.
- Jobs: identify the active model and project, see whether work is progressing, anticipate context pressure, detect stale or inferred data, and inspect shared task state from either runtime's shell without claiming native Codex injection.

## Information architecture

Priority order is fixed by decision value, not by data availability:

1. Model and reasoning effort
2. Project and branch
3. Task and phase
4. Progress and run state
5. Context pressure and forecast
6. Optional cost or quota

Identity, email, account plan, session ID, CLI version, clock, and weather are excluded from the default rail.

## Selected UI: Signal Rail

Wide (160+ columns):

```text
◆ GPT-5.5 · high  ◇ signalrail/main  ▶ Build renderer  3/7 ███░░░░  CTX 38% left ~42m  $1.24~
```

Standard (100-159 columns):

```text
◆ GPT-5.5 high  ◇ signalrail/main  ▶ Build renderer  3/7  CTX 38% left
```

Compact (60-99 columns):

```text
GPT-5.5  signalrail  Build renderer  3/7  C38%
```

Minimal (<60 columns):

```text
GPT-5.5  Build renderer  C38%
```

Markers:

- No suffix: exact and fresh
- `~`: estimated
- `↻`: cached but within configured TTL
- `!`: stale or degraded
- `—`: unavailable only when the field is pinned by the user; otherwise omit it

The renderer allocates an information budget from left to right. It first shortens labels, then removes decorations, then drops low-priority segments. It never slices ANSI sequences or wraps.

## Rejected UI options

### Focus Spine

```text
GPT-5.5 high │ signalrail:main │ Build renderer 3/7 │ ctx 38% left │ $1.24 est
```

Clear and portable, but visually flat and too similar to existing delimiter-based status lines.

### Dual Deck

```text
MODEL GPT-5.5 high   PROJECT signalrail/main   TASK Build renderer
PROGRESS 3/7 █████░░   CONTEXT 38% left ~42m   COST $1.24 est
```

Best for dashboards, but consumes terminal height and is more prone to redraw glitches. It remains an opt-in Claude-only layout after v1.

## Design principles

- Decision density over information density.
- Honest uncertainty over polished precision.
- Shared semantics, runtime-specific integrations.
- Fast and local by default.
- Graceful degradation before customization.

## Visual language

- Color: model cyan, project blue, active task amber, healthy progress green, context pressure amber/red; neutral text remains theme-derived.
- Typography: terminal-native monospace only; no bundled font.
- Spacing: one cell inside semantic groups and two cells between groups when width permits.
- Shape: lightweight glyph anchors, no filled powerline blocks by default.
- Motion: none in the status line; activity is expressed as a stable verb or state.
- Iconography: Unicode geometric marks with an ASCII fallback; Nerd Fonts are optional, never required.

## Components

- Normalized snapshot: runtime-neutral model for session, project, task, progress, context, cost, quota, freshness, and provenance.
- Claude adapter: converts official stdin JSON to a normalized snapshot.
- Codex compiler: maps SignalRail segment intent to supported native `[tui].status_line`, color, and terminal-title settings.
- Project state: `.signalrail/state.json` stores task, phase, progress, blockers, and timestamps for handoff through SignalRail commands and the Claude renderer. Codex's native rail cannot read this file.
- Config resolver: built-in defaults, user config, project config, runtime profile, then CLI flags.
- Adaptive renderer: plain, ANSI, and JSON outputs with width-aware segment selection.
- Settings: bilingual interactive menu; persisted labels and status-line output remain English.
- Diagnostics: capability matrix, data provenance, freshness, and install checks.

## Runtime capability contract

Claude Code provides command execution, JSON stdin, ANSI, links, multi-line output, and periodic refresh. SignalRail uses the full renderer there.

Codex currently provides configurable built-in status items, theme-derived colors, and terminal-title items, but no external command provider. SignalRail therefore compiles the shared intent into the closest native Codex configuration. Codex's `task-progress` reflects Codex `update_plan`, not `.signalrail/state.json`. Shared project task state remains available through `signalrail task show` and `signalrail preview`. Unsupported custom segments produce an explicit warning.

The v1 compatibility baseline is OpenAI Codex `main` commit `406062c3af8b27c8e1b4b83c485ebe1ae0df874c` (2026-06-19). Generated items are limited to: `model`, `model-with-reasoning`, `reasoning`, `current-dir`, `project-name`, `git-branch`, `run-state`, `context-remaining`, `context-used`, `five-hour-limit`, `weekly-limit`, `codex-version`, `context-window-size`, `used-tokens`, `thread-id`, `fast-mode`, `thread-title`, and `task-progress`.

Terminal titles use a separate enum pinned to the same commit: `app-name`, `project-name`, `current-dir`, `activity`, `run-state`, `thread-title`, `git-branch`, `context-remaining`, `context-used`, `five-hour-limit`, `weekly-limit`, `codex-version`, `used-tokens`, `total-input-tokens`, `total-output-tokens`, `thread-id`, `fast-mode`, `model`, `model-with-reasoning`, `reasoning`, and `task-progress`.

## Configuration

Resolution order:

1. Built-in defaults
2. `$XDG_CONFIG_HOME/signalrail/config.toml` or `~/.config/signalrail/config.toml`
3. `<project>/.signalrail.toml`
4. `[runtime.claude]` or `[runtime.codex]`
5. CLI flags

Arrays replace lower-priority arrays. Tables merge by key. Unknown keys are rejected. Project files may be committed; mutable state is ignored by default.

## Accessibility and compatibility

- ANSI color is supplementary; labels and order carry meaning without it.
- `NO_COLOR`, `TERM=dumb`, ASCII mode, and no-Nerd-Font mode are supported.
- Contrast targets WCAG AA-equivalent terminal palettes where controllable.
- No animation is required to understand state.
- v1 targets macOS and Linux; Windows is covered in CI and documented as best effort.

## Responsive behavior

- 160+ cells: all configured core segments and optional cost.
- 100-159 cells: remove forecast detail and decorative meters first.
- 60-99 cells: shorten labels, project paths, task text, and context notation.
- Below 60 cells: preserve model, task, and context in that order.
- Mouse and touch interactions are out of scope; all configuration is keyboard operable.

## Interaction states

- Loading: render available high-priority fields; never block waiting for optional providers.
- Empty: show model and project if known, otherwise a terse diagnostic in preview mode only.
- Error: protocol stdout stays clean; diagnostics go to stderr.
- Stale: keep the last value with `!` and age in explain mode.
- Degraded runtime: map supported fields and report omitted capabilities.

## Content voice

- Status line: English, terse nouns and verbs, no sentences.
- Settings: English and Simplified Chinese selected on first run or with `--lang`.
- Warnings: state impact first, then one action.

## Implementation constraints

- Go 1.23+ single binary.
- Rendering path performs no network requests.
- Cached render target: p95 under 20 ms; cold render target: p95 under 50 ms on supported macOS/Linux hardware.
- All terminal width calculations use display-cell width, not byte or rune count.
- Installer writes backups and supports `--dry-run`.
- No global config mutation occurs during tests.

## v1 delivery boundary

v1 includes one-line rendering, Claude stdin integration, Codex native-config generation, project/user TOML, project task state, bilingual numbered settings, preview, explain, doctor, safe installers, tests, CI, and source-level release documentation.

Deferred beyond v1: Dual Deck rendering, network quota fetches, background daemons, transcript watchers, native live Codex custom text, IDE extensions, package-manager publication, signed binaries, and web dashboards.

## Differentiating features

- Confidence-aware metrics with separate confidence (`exact`, `estimated`, `unavailable`) and freshness (`fresh`, `cached`, `stale`, `degraded`) dimensions.
- Adaptive Information Budget instead of fixed breakpoint templates.
- Context forecast based on recent consumption rate, clearly marked estimated.
- Session Pulse for active, idle, blocked, and stale work.
- Cross-agent task handoff through project-local state.
- Explain mode showing why every segment is present, hidden, or degraded.
- Privacy mode that removes usernames, home paths, remote owners, and sensitive branch patterns.

## Success signals

- No wrapping or broken ANSI at 40, 60, 80, 120, and 160 columns.
- Project task state is visible in Claude and in runtime-neutral `task show/preview`; Codex native `task-progress` truthfully represents only Codex `update_plan`.
- Every non-exact metric exposes provenance and freshness in `explain --json`.
- Claude install works from official stdin fixtures; Codex output uses only the pinned compatibility item IDs above.
- `render` exits 0 with one protocol-clean line, invalid input exits 2 with stderr diagnostics, and unsupported Codex fields warn or fail according to policy.
- Cached render p95 is under 20 ms and cold render p95 is under 50 ms in the release benchmark.
- Privacy tests prove home paths, usernames, and remote owners are redacted; provenance tests cover every rendered segment.
- Full test, lint, build, and install smoke checks pass on release branches.

## Open questions

- [ ] Promote the optional Dual Deck layout after real-world redraw testing.
- [ ] Add a documented Codex live-render adapter if upstream lands command-backed status-line support.
- [ ] Add signed Homebrew and Scoop release channels after the first tagged release.
