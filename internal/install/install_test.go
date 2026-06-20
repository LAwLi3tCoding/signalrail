package install

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPlanClaudePreservesUnrelatedSettingsAndStatusOptions(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	before := `{"permissions":{"allow":["Bash(git:*)"]},"statusLine":{"refreshInterval":7,"hideVimModeIndicator":true,"command":"old"}}`
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	change, err := PlanClaude(path, "signalrail render --runtime claude")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatal("planning wrote file")
	}
	var got map[string]any
	if err := json.Unmarshal(change.After, &got); err != nil {
		t.Fatal(err)
	}
	if got["permissions"] == nil {
		t.Fatal("permissions lost")
	}
	line := got["statusLine"].(map[string]any)
	if line["command"] != "signalrail render --runtime claude" || line["type"] != "command" || line["refreshInterval"] != float64(7) || line["hideVimModeIndicator"] != true {
		t.Fatalf("statusLine=%v", line)
	}
	again, err := PlanClaudeBytes(path, change.After, "signalrail render --runtime claude")
	if err != nil || string(again.After) != string(change.After) {
		t.Fatalf("not idempotent: err=%v", err)
	}
}

func TestPlanCodexPreservesCommentsAndMapsSupportedIntent(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	before := "model = \"gpt-5.5\"\n\n[tui]\n# keep this comment\nnotifications = true\nstatus_line = [\"old\"]\n\n[history]\npersistence = \"save-all\"\n"
	if err := os.WriteFile(path, []byte(before), 0o600); err != nil {
		t.Fatal(err)
	}
	change, warnings, err := PlanCodex(path, []string{"model", "project", "branch", "progress", "context", "cost"}, []string{"activity", "project"})
	if err != nil {
		t.Fatal(err)
	}
	after := string(change.After)
	for _, required := range []string{"# keep this comment", "# SignalRail managed status line", "notifications = true", "[history]", "model-with-reasoning", "project-name", "git-branch", "task-progress", "context-remaining", "status_line_use_colors = true"} {
		if !strings.Contains(after, required) {
			t.Fatalf("missing %q in:\n%s", required, after)
		}
	}
	if strings.Contains(after, `"cost"`) || len(warnings) != 1 {
		t.Fatalf("warnings=%v after=%s", warnings, after)
	}
	again, _, err := PlanCodexBytes(path, change.After, []string{"model", "project", "branch", "progress", "context", "cost"}, []string{"activity", "project"})
	if err != nil || string(again.After) != after {
		t.Fatalf("not idempotent: err=%v\n%s", err, again.After)
	}
}

func TestPlanCodexPreservesUserTerminalTitleWhenUnmanaged(t *testing.T) {
	before := []byte("[tui]\nterminal_title = [\"app-name\", \"git-branch\"]\n")
	change, warnings, err := PlanCodexBytes("config.toml", before, []string{"model"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("warnings=%v", warnings)
	}
	if !strings.Contains(string(change.After), `terminal_title = ["app-name", "git-branch"]`) {
		t.Fatalf("user terminal title was removed:\n%s", change.After)
	}
}

func TestMalformedConfigDoesNotPlanChanges(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "settings.json")
	_ = os.WriteFile(jsonPath, []byte("{"), 0o600)
	if _, err := PlanClaude(jsonPath, "signalrail"); err == nil {
		t.Fatal("expected JSON error")
	}
	tomlPath := filepath.Join(dir, "config.toml")
	_ = os.WriteFile(tomlPath, []byte("[tui\n"), 0o600)
	if _, _, err := PlanCodex(tomlPath, []string{"model"}, nil); err == nil {
		t.Fatal("expected TOML error")
	}
}

func TestApplyCreatesBackupAndWritesAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	change, err := PlanClaude(path, "signalrail")
	if err != nil {
		t.Fatal(err)
	}
	if err := change.Apply(true); err != nil {
		t.Fatal(err)
	}
	backups, _ := filepath.Glob(path + ".bak.*")
	if len(backups) != 1 {
		t.Fatalf("backups=%v", backups)
	}
	if data, _ := os.ReadFile(backups[0]); string(data) != `{"theme":"dark"}` {
		t.Fatalf("backup=%q", data)
	}
	if data, _ := os.ReadFile(path); string(data) != string(change.After) {
		t.Fatalf("written=%q", data)
	}
	if err := change.Apply(true); err != nil {
		t.Fatal(err)
	}
	backups, _ = filepath.Glob(path + ".bak.*")
	if len(backups) != 1 {
		t.Fatalf("idempotent backups=%v", backups)
	}
}

func TestApplyRejectsStalePlanAndPreservesMode(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"theme":"dark"}`), 0o640); err != nil {
		t.Fatal(err)
	}
	change, err := PlanClaude(path, "signalrail")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(`{"theme":"light"}`), 0o640); err != nil {
		t.Fatal(err)
	}
	if err := change.Apply(true); !errors.Is(err, ErrStalePlan) {
		t.Fatalf("err=%v", err)
	}
	if data, _ := os.ReadFile(path); string(data) != `{"theme":"light"}` {
		t.Fatalf("stale plan overwrote file: %s", data)
	}
	change, err = PlanClaude(path, "signalrail")
	if err != nil {
		t.Fatal(err)
	}
	if err := change.Apply(false); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o640 {
		t.Fatalf("mode=%o", info.Mode().Perm())
	}
}

func TestPlanningRejectsSymlinkAndFailedRenameCleansTemporaryFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.json")
	link := filepath.Join(dir, "settings.json")
	if err := os.WriteFile(target, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := PlanClaude(link, "signalrail"); err == nil {
		t.Fatal("expected symlink rejection")
	}
	change, err := PlanClaude(target, "signalrail")
	if err != nil {
		t.Fatal(err)
	}
	err = change.apply(false, func(string, string) error { return errors.New("rename failed") })
	if err == nil {
		t.Fatal("expected rename failure")
	}
	if data, _ := os.ReadFile(target); string(data) != `{}` {
		t.Fatalf("target changed: %s", data)
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".signalrail-*.tmp"))
	if len(matches) != 0 {
		t.Fatalf("temps=%v", matches)
	}
}

func TestCodexWarningsAreCapabilitySpecificAndTitlesValidated(t *testing.T) {
	change, warnings, err := PlanCodexBytes("config.toml", nil, []string{"cost", "forecast", "task", "custom-task-text"}, []string{"activity", "project", "bogus"})
	if err != nil {
		t.Fatal(err)
	}
	codes := map[string]bool{}
	for _, warning := range warnings {
		codes[warning.Code] = true
	}
	for _, code := range []string{"codex-cost-unsupported", "codex-forecast-unsupported", "codex-task-state-unsupported", "codex-custom-task-unsupported", "unsupported-codex-title"} {
		if !codes[code] {
			t.Fatalf("missing %s in %v", code, warnings)
		}
	}
	text := string(change.After)
	if strings.Contains(text, "bogus") || !strings.Contains(text, `terminal_title = ["activity", "project-name"]`) {
		t.Fatalf("config=%s", text)
	}
}
