# SignalRail Onboarding Reliability Design

## Decision

Improve first-run reliability without adding a multi-file `setup` transaction. Existing atomic commands remain the stable interface.

## Scope

1. Change the module and repository identity to `github.com/LAwLi3tCoding/signalrail` everywhere.
2. Make effective configuration authoritative for default preview and Codex installation.
3. Split documentation into Claude, Codex, and combined usage tracks.
4. Make CLI help complete and doctor warnings actionable.

## Behavior

`signalrail preview` loads user, project, and Claude runtime configuration. An explicit `--preset` keeps the existing static sample behavior. `--runtime`, `--project`, and `--home` select the effective configuration source.

`signalrail install codex` derives native intent from effective Codex segments when `--items` is absent. Project intent includes project name and branch. `--items` remains an invocation-only override with highest priority.

Doctor warnings include the exact install command for the missing integration. Codex text continues to state that native `task-progress` is owned by Codex and cannot read SignalRail task state.

## Compatibility

- Configuration schema stays at version 1.
- Existing explicit preview presets and install items remain unchanged.
- Installer backup, stale-plan, symlink, and atomic-write behavior is unchanged.
- No command-backed Codex renderer is introduced.

## Deferred

- A one-command setup flow, until cross-file transaction and rollback semantics exist.
- Tagged releases and package-manager distribution, which require an explicit release decision.
- Forecast, quota, and session-state expansion.

## Acceptance

- The documented `go install` path matches the public repository and module.
- Effective project/runtime segments control default preview and default Codex dry-run output.
- Explicit `--preset` and `--items` override effective configuration.
- Help documents every supported preview and install option.
- Missing integrations produce copyable doctor remediation commands.
- Race tests, smoke checks, and macOS/Linux/Windows CI pass.
