# SignalRail Onboarding Reliability Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make SignalRail installable from its real repository and make first-run commands honor one effective configuration.

**Architecture:** Keep config resolution in `internal/config`, runtime compilation in `internal/install`, and orchestration in `internal/cli`. Preserve every explicit override and all safe-write guarantees.

**Tech Stack:** Go 1.23+, standard library, TOML v2, GitHub Actions.

## Global Constraints

- Module identity is `github.com/LAwLi3tCoding/signalrail`.
- Claude retains full command rendering; Codex remains native-item-only.
- Configuration schema remains version 1.
- Every behavior change begins with a failing test.
- Existing installer safety and exit codes remain unchanged.

---

### Task 1: Correct repository identity

**Files:**
- Modify: `go.mod`, Go imports, `README.md`, `CONTRIBUTING.md`
- Test: all Go packages

**Interfaces:**
- Produces the canonical module path used by source imports and install docs.

- [ ] Replace every `github.com/LAwLi3tCoding/signalrail` reference with `github.com/LAwLi3tCoding/signalrail`.
- [ ] Run `go mod tidy` and `go list -m`; expect the canonical module path.
- [ ] Run `go test ./...`; expect PASS.
- [ ] Commit `fix: correct SignalRail module identity`.

### Task 2: Use effective configuration

**Files:**
- Modify: `internal/cli/app.go`
- Test: `internal/cli/app_test.go`
- Update: `scripts/smoke.sh`, `scripts/smoke.ps1`

**Interfaces:**
- `preview` consumes `config.Load(project, home, runtime, overrides)` unless `--preset` is explicit.
- Codex install converts `[]status.SegmentName` to native intent when `--items` is absent.

- [ ] Add a failing test where project segments `["model", "context"]` control plain `preview` output.
- [ ] Add a failing test where `[runtime.codex]` controls default Codex dry-run output.
- [ ] Add regression tests proving explicit `--preset` and `--items` still override configuration.
- [ ] Add preview options `--runtime`, `--project`, and `--home`; reject invalid runtimes.
- [ ] Map project intent to both `project` and `branch`; pass task/cost through for explicit capability warnings.
- [ ] Update both smoke scripts to exercise effective-config preview and Codex compilation.
- [ ] Run `go test -race ./internal/cli ./internal/config ./internal/install`; expect PASS.
- [ ] Commit `feat: apply effective config to onboarding commands`.

### Task 3: Make onboarding actionable

**Files:**
- Modify: `README.md`, `internal/cli/app.go`, `internal/diagnostics/diagnostics.go`
- Test: `internal/cli/app_test.go`

**Interfaces:**
- Help remains protocol-clean stdout.
- Doctor keeps exit codes 0 healthy, 1 actionable warning, and 2 invalid configuration.

- [ ] Add failing tests for complete preview/install help and doctor remediation text.
- [ ] Expand help with `--runtime`, `--project`, `--home`, `--items`, and `--strict`.
- [ ] Add exact Claude and Codex install commands to missing-integration warnings.
- [ ] Rewrite Quick start as Claude Code, Codex, and Both tracks; explain user versus project scope.
- [ ] Run CLI tests and smoke checks; expect PASS.
- [ ] Commit `docs: make first-run workflow actionable`.

### Task 4: Review and publish

**Files:** Modify only for confirmed review findings.

- [ ] Request an independent review against the design and full optimization diff.
- [ ] Fix every confirmed issue with a regression test.
- [ ] Run format, mod verify, vet, race tests, benchmark, build, smoke, diff, and secret checks.
- [ ] Update `codex/signalrail`, confirm the remote tree equals local HEAD, and wait for all CI checks.
- [ ] Record exact usage commands and the Codex limitation in the handoff.

## Self-review

- All design acceptance criteria map to Tasks 1-4.
- No placeholder or schema change remains.
- Explicit overrides retain backward-compatible precedence.
