# Competitive Landscape — 2026-06-19

This survey uses repository metadata and upstream documentation available on 2026-06-19. Stars are a discovery signal, not a quality score.

| Project | Signal | Strength | Gap relevant to SignalRail |
|---|---:|---|---|
| [CCometixLine](https://github.com/Haleclipse/CCometixLine) | 3,191 stars, Rust | Fast binary, Git/context integration, TUI themes | Claude-specific; emphasizes customization over data confidence |
| [nilbuild/claude-statusline](https://github.com/nilbuild/claude-statusline) | 1,294 stars, shell | Minimal, easy to inspect and install | Narrow scope; limited cross-runtime/state model |
| [rz1989s/claude-code-statusline](https://github.com/rz1989s/claude-code-statusline) | 457 stars, shell | Many widgets and themes | Information density and external widgets can distract from session decisions |
| [ccstatusline](https://github.com/sirmalloc/ccstatusline) | active TypeScript | Rich widget catalog, powerline, interactive TUI, localization | Node/Bun runtime, Claude-only command contract, configuration surface is large |
| [claude-code-statusline-pro](https://github.com/Wangnov/claude-code-statusline-pro) | 225 stars, Rust | Native performance and polished Claude status line | Provider-specific and not task/provenance centered |
| [claude-pace](https://github.com/Astro-Han/claude-pace) | 208 stars, shell | Pace-aware quota monitoring | Specialized quota tool rather than full workflow state |
| [claude-code-usage-bar](https://github.com/leeguooooo/claude-code-usage-bar) | 279 stars, Python | Usage/reset visibility and compact styles | Runtime and quota specific; daemon mode adds operational complexity |
| [ilia-pluzhnikov/claude-code-statusline](https://github.com/ilia-pluzhnikov/claude-code-statusline) | 70 stars, JavaScript | Dependency-free, includes task and peak-hours data | Claude-specific, inferred data is not a first-class UI concept |
| [syou6162/ccstatusline](https://github.com/syou6162/ccstatusline) | Go | YAML templates, command actions, TTL cache, XDG paths | Arbitrary shell execution expands latency and security surface |
| [codex-statusline](https://github.com/rgomes87/codex-statusline) | shell/tmux | Works around Codex limits with a separate tmux area | Requires tmux and does not integrate with Codex's supported TUI contract |
| [codex-statusline-vscode](https://github.com/jacsuper/codex-statusline-vscode) | TypeScript/VS Code | Watches Codex session logs and exposes live state | IDE-specific and depends on internal log behavior |
| [OpenAI Codex](https://github.com/openai/codex) | upstream | Native model, project, task progress, context, limits, colors, terminal title | No external command-backed status renderer as of this survey |

## Upstream capability evidence

- [Claude Code status-line documentation](https://code.claude.com/docs/en/statusline) defines command JSON stdin, ANSI/multi-line output, event updates, refresh intervals, context, cost, rate limits, task-related subagent data, and trust requirements.
- [OpenAI Codex config schema](https://github.com/openai/codex/blob/main/codex-rs/core/config.schema.json) defines native `tui.status_line`, `status_line_use_colors`, and `terminal_title` configuration.
- [OpenAI Codex issue #20043](https://github.com/openai/codex/issues/20043) documents the missing command-backed extension and was closed as a duplicate of the active customization request.
- [OpenAI Codex issue #10233](https://github.com/openai/codex/issues/10233) documents the lack of a supported headless equivalent for interactive status data.

## Market pattern

Most products optimize one of four axes: more widgets, more themes, lower startup cost, or better quota visibility. Few make uncertainty, provenance, current task, and runtime capability differences part of the product model. SignalRail targets that gap.

## Independent design boundary

SignalRail does not copy competitor source, layouts, names, or configuration schemas. Reused ideas are generic terminal conventions: ordered segments, ANSI color, adaptive width, local caching, project configuration, and interactive setup. Its defining model—decision-priority layout, confidence/freshness metadata, cross-agent task state, and an honest Codex compiler—is independently specified in `DESIGN.md`.
