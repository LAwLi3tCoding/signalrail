# SignalRail Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development or superpowers:executing-plans. Steps use checkbox syntax for tracking.

**Goal:** Build and publish a fast cross-runtime status-line CLI with a full Claude Code renderer and an honest Codex native-config compiler.

**Architecture:** A Go binary normalizes runtime input into provenance-aware snapshots, resolves project/user configuration, and renders a width-adaptive rail. Claude Code invokes the renderer; Codex receives deterministic native TUI configuration because it has no command-backed status-line API.

**Tech Stack:** Go 1.23+, standard library, `go-runewidth`, TOML v2, GitHub Actions.

## Global Constraints

- Status-line copy is English; interactive settings support `en` and `zh-CN`.
- Rendering performs no network access and keeps protocol stdout diagnostic-free.
- Arrays replace lower config layers; tables merge; unknown keys fail validation.
- No wrap or broken ANSI at 40, 60, 80, 120, or 160 cells.
- Confidence (`exact`, `estimated`, `unavailable`) and freshness (`fresh`, `cached`, `stale`, `degraded`) remain separate and visible.
- Codex capability gaps are reported, never hidden.
- Mutable state stays in `.signalrail/state.json`; project policy stays in `.signalrail.toml`.

## File map

- `cmd/signalrail/main.go`: process entry point.
- `internal/cli/app.go`: command routing and stdout/stderr discipline.
- `internal/status/model.go`: normalized snapshot and provenance types.
- `internal/config/{model,load}.go`: defaults, TOML loading, merge, validation.
- `internal/adapter/claude.go`: official Claude JSON normalization.
- `internal/render/{render,width}.go`: segments, ANSI, adaptive budget.
- `internal/task/store.go`: project task handoff state.
- `internal/install/{claude,codex}.go`: safe config edits and backups.
- `internal/diagnostics/diagnostics.go`: doctor and explain reports.
- `testdata/`: sanitized contracts and golden output.

---

### Task 1: Core model and configuration

**Files:** Create `go.mod`, `internal/status/model.go`, `internal/config/model.go`, `internal/config/load.go`; test `internal/config/load_test.go`.

**Interfaces:**
- Produces `status.Datum[T] { Value, Confidence, Freshness, Source, ObservedAt, ExpiresAt }`.
- Produces `config.Load(projectDir, userDir string, runtime Runtime, overrides Overrides) (Config, error)`.

- [ ] Write tests for defaults, user/project/runtime precedence, array replacement, and unknown-key errors.
- [ ] Run `go test ./internal/config`; expect failure because packages are absent.
- [ ] Define provenance, freshness, snapshot, segment, privacy, task, context, and cost types.
- [ ] Implement versioned config defaults and strict TOML decoding.
- [ ] Run focused tests, then `go test ./...`; expect PASS.
- [ ] Commit `feat: add status model and config resolution`.

### Task 2: Task handoff state

**Files:** Create `internal/task/store.go`, `internal/task/store_test.go`.

**Interfaces:**
- `Load(root string) (status.Task, error)`
- `Update(root string, mutation Mutation, now time.Time) (status.Task, error)`
- `Mutation` supports set, step, block, done, clear.

- [ ] Write state-transition and atomic-write tests using `t.TempDir()`.
- [ ] Run focused tests; expect missing API failure.
- [ ] Implement project-root discovery, validation, atomic rename, and stable JSON.
- [ ] Verify clear removes state and block retains progress.
- [ ] Run `go test -race ./internal/task`; expect PASS.
- [ ] Commit `feat: add cross-agent task state`.

### Task 3: Claude adapter and providers

**Files:** Create `internal/adapter/claude.go`, `internal/adapter/claude_test.go`, `internal/provider/git.go`, `internal/provider/git_test.go`, `testdata/claude/*.json`.

**Interfaces:**
- `adapter.ParseClaude(io.Reader, time.Time) (status.Snapshot, error)`.
- `provider.Git(ctx, cwd) status.Project` returns cached/degraded metadata without blocking rendering.

- [ ] Add official-shaped full, partial, null, malformed, and future-field fixtures.
- [ ] Write parser tests proving client cost is estimated and context input is exact.
- [ ] Run tests; expect missing parser failure.
- [ ] Implement bounded JSON decoding with forward-compatible unknown fields.
- [ ] Add Git branch/dirty tests in temporary repositories.
- [ ] Implement context-bounded Git commands with no optional locks.
- [ ] Run adapter/provider tests with race detection; expect PASS.
- [ ] Commit `feat: normalize Claude and project status`.

### Task 4: Adaptive Signal Rail renderer

**Files:** Create `internal/render/render.go`, `internal/render/width.go`, tests and `testdata/golden/*.txt`.

**Interfaces:**
- `Render(snapshot, config, Options{Width, Format, Color}) (Result, error)`.
- `Result` includes output plus included, shortened, and omitted segment explanations.

- [ ] Write golden tests at 40/60/80/120/160 cells and ASCII/NO_COLOR modes.
- [ ] Add tests for CJK width, emoji, ANSI stripping, confidence suffixes, and privacy redaction.
- [ ] Run focused tests; expect missing renderer failure.
- [ ] Implement semantic segments and deterministic shorten/drop phases.
- [ ] Assert every output is one line and visible width never exceeds budget.
- [ ] Run `go test -race ./internal/render`; expect PASS.
- [ ] Commit `feat: render adaptive Signal Rail UI`.

### Task 5: Safe runtime installers

**Files:** Create `internal/install/claude.go`, `internal/install/codex.go`, `internal/install/install_test.go`.

**Interfaces:**
- `PlanClaude(path, command string) (Change, error)` preserves unrelated JSON.
- `PlanCodex(path string, items, title []string) (Change, []Warning, error)` edits only `[tui]` owned keys.
- `Change.Apply(backup bool) error` uses atomic writes.

- [ ] Write dry-run, backup, idempotency, comments-preserved, and malformed-config tests.
- [ ] Run tests; expect missing planner failure.
- [ ] Implement Claude `statusLine` merge with `padding=0` and configurable refresh.
- [ ] Implement the pinned Codex item mapping from `DESIGN.md`; document that native `task-progress` comes from Codex `update_plan` only.
- [ ] Emit warnings for cost, forecast, SignalRail task state, and custom task text unsupported by Codex.
- [ ] Run installer tests with race detection; expect PASS.
- [ ] Commit `feat: add Claude and Codex installers`.

### Task 6: CLI, bilingual settings, explain, doctor

**Files:** Create `internal/cli/app.go`, `internal/cli/app_test.go`, `internal/diagnostics/diagnostics.go`, `cmd/signalrail/main.go`.

**Interfaces:** `cli.Run(ctx, args, stdin, stdout, stderr) int`.

- [ ] Write CLI tests for every required command, exit code, and stdout/stderr boundary.
- [ ] Run CLI tests; expect missing router failure.
- [ ] Implement render, preview, task, install, explain, doctor, and version commands.
- [ ] Implement keyboard-only numbered settings menu with English and Simplified Chinese catalogs.
- [ ] Ensure settings write valid TOML and preview before save.
- [ ] Run `go test -race ./...`; expect PASS.
- [ ] Commit `feat: add SignalRail command interface`.

### Task 7: Documentation, CI, and release readiness

**Files:** Create `README.md`, `LICENSE`, `CONTRIBUTING.md`, `.github/workflows/ci.yml`, `examples/*.toml`, `scripts/smoke.sh`; update `DESIGN.md` only for verified implementation decisions.

- [ ] Write README acceptance checks for install, Claude, Codex, task handoff, privacy, and troubleshooting.
- [ ] Add MIT license and contribution/test instructions.
- [ ] Add Linux/macOS/Windows CI for format, vet, race tests, and build.
- [ ] Add smoke script that builds a clean binary, renders fixtures, and validates generated config.
- [ ] Run `gofmt -w`, `go vet ./...`, `go test -race ./...`, `go build ./cmd/signalrail`, and smoke tests.
- [ ] Commit `docs: prepare SignalRail for release`.

### Task 8: Independent verification and GitHub publication

**Files:** Modify only files required by reviewer findings.

- [ ] Dispatch a code-reviewer against requirements, design, and full diff.
- [ ] Fix every confirmed high/medium issue using a failing regression test first.
- [ ] Re-run format, vet, race tests, build, smoke, `git diff --check`, and secret scan.
- [ ] Confirm intended files only with `git status -sb` and `git diff main...HEAD --stat`.
- [ ] Create GitHub repository if no remote exists, push `codex/signalrail`, and open a draft PR.
- [ ] Record branch, commit, PR URL, exact verification commands, and known Codex limitation.

## Plan self-review

- Every design requirement maps to Tasks 1-7.
- Runtime asymmetry is handled in Task 5 rather than hidden in rendering.
- Every production behavior begins with a failing test.
- No task requires global config writes during tests.
- Publication occurs only after independent review and fresh verification.
