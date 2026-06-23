#!/usr/bin/env bash
set -euo pipefail

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT

cd "$repo_root"
go build -o "$tmp/signalrail" ./cmd/signalrail

for preset in wide standard compact minimal; do
  output="$($tmp/signalrail preview --preset "$preset")"
  test -n "$output"
  test "$(printf '%s' "$output" | wc -l | tr -d ' ')" -eq 0
done

mkdir -p "$tmp/project/.git"
printf '2\ny\n' | "$tmp/signalrail" config --lang en --scope project --project "$tmp/project" --home "$tmp/home" >/dev/null
test -f "$tmp/project/.signalrail.toml"
cat >"$tmp/project/.signalrail.toml" <<'EOF'
version = 1
segments = ["model", "context"]

[runtime.codex]
segments = ["model", "context"]
EOF
"$tmp/signalrail" explain --json --project "$tmp/project" --home "$tmp/home" >/dev/null
effective_preview="$($tmp/signalrail preview --project "$tmp/project" --home "$tmp/home")"
printf '%s' "$effective_preview" | grep -q 'GPT-5.5'
printf '%s' "$effective_preview" | grep -q 'CTX 38% left'
if printf '%s' "$effective_preview" | grep -q 'Build renderer'; then
  echo "effective preview ignored configured segments" >&2
  exit 1
fi
"$tmp/signalrail" install codex --scope project --dry-run --home "$tmp/home" --project "$tmp/project" >"$tmp/codex-dry-run.toml"
grep -q 'status_line = \["model-with-reasoning", "context-remaining"\]' "$tmp/codex-dry-run.toml"
if grep -q 'git-branch' "$tmp/codex-dry-run.toml"; then
  echo "Codex dry-run ignored configured runtime segments" >&2
  exit 1
fi

"$tmp/signalrail" task set "Smoke task" --total 2 --project "$tmp/project" >/dev/null
"$tmp/signalrail" task step --project "$tmp/project" >/dev/null
"$tmp/signalrail" task show --project "$tmp/project" | grep -q '"Step": 1'

"$tmp/signalrail" install claude --scope project --home "$tmp/home" --project "$tmp/project" >/dev/null
"$tmp/signalrail" install codex --scope project --home "$tmp/home" --project "$tmp/project" >/dev/null
grep -q 'signalrail render' "$tmp/project/.claude/settings.json"
grep -q 'SignalRail managed status line' "$tmp/project/.codex/config.toml"
"$tmp/signalrail" doctor --json --home "$tmp/home" --project "$tmp/project" >/dev/null

cat testdata/claude/full.json | "$tmp/signalrail" render --runtime claude --width 80 --project "$tmp/project" --home "$tmp/home" | grep -q 'Opus 4.7'

echo "SignalRail smoke checks passed"
