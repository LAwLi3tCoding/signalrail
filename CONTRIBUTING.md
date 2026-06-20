# Contributing

SignalRail favors small, evidence-backed changes. Before changing behavior:

1. Read `DESIGN.md` and the relevant specification.
2. Add a failing behavior test and confirm the expected failure.
3. Implement the smallest change that passes.
4. Run format, vet, race tests, build, and smoke checks.
5. Update capability documentation when a runtime contract changes.

Do not add network calls, identity fields, arbitrary segment shell execution, or undocumented Codex integrations without a reviewed design change.

Pull requests should explain the user impact, runtime boundary, validation performed, and whether output or configuration compatibility changes.
