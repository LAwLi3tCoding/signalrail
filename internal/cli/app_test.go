package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/LAwLi3t-CN/signalrail/internal/adapter"
	"github.com/LAwLi3t-CN/signalrail/internal/config"
	statusrender "github.com/LAwLi3t-CN/signalrail/internal/render"
)

func TestRenderAndMalformedProtocolBoundaries(t *testing.T) {
	fixture, err := os.ReadFile("../../testdata/claude/full.json")
	if err != nil {
		t.Fatal(err)
	}
	root := initProject(t)
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"render", "--runtime", "claude", "--width", "80", "--project", root, "--home", t.TempDir()}, bytes.NewReader(fixture), &out, &stderr)
	if code != 0 || stderr.Len() != 0 || strings.Count(strings.TrimSuffix(out.String(), "\n"), "\n") != 0 || !strings.Contains(out.String(), "Opus 4.7") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"render", "--runtime", "claude", "--project", root, "--home", t.TempDir()}, strings.NewReader("{"), &out, &stderr)
	if code != 2 || out.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
}

func TestPreviewUnknownPresetWritesNothing(t *testing.T) {
	root := initProject(t)
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"preview", "--preset", "unknown", "--project", root}, strings.NewReader(""), &out, &stderr)
	if code != 2 || out.Len() != 0 || stderr.Len() == 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".signalrail")); !os.IsNotExist(err) {
		t.Fatalf("preview wrote state: %v", err)
	}
}

func TestTaskCommandsAndInvalidTransition(t *testing.T) {
	root := initProject(t)
	run := func(args ...string) (int, string, string) {
		var out, err bytes.Buffer
		all := append([]string{"task"}, args...)
		all = append(all, "--project", root)
		code := Run(context.Background(), all, strings.NewReader(""), &out, &err)
		return code, out.String(), err.String()
	}
	if code, _, err := run("set", "Build renderer", "--total", "2"); code != 0 {
		t.Fatalf("set: %d %s", code, err)
	}
	if code, _, err := run("step"); code != 0 {
		t.Fatalf("step: %d %s", code, err)
	}
	if code, out, _ := run("show"); code != 0 || !strings.Contains(out, `"Step": 1`) {
		t.Fatalf("show: %d %s", code, out)
	}
	if code, _, err := run("done"); code != 0 {
		t.Fatalf("done: %d %s", code, err)
	}
	if code, _, _ := run("step"); code != 2 {
		t.Fatalf("completed step code=%d", code)
	}
	if code, _, err := run("clear"); code != 0 {
		t.Fatalf("clear: %d %s", code, err)
	}
}

func TestChineseSettingsWritesEnglishTOMLAndCancelDoesNotWrite(t *testing.T) {
	root := initProject(t)
	path := filepath.Join(root, ".signalrail.toml")
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"config", "--lang", "zh-CN", "--scope", "project", "--project", root}, strings.NewReader("2\ny\n"), &out, &stderr)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(out.String(), "SignalRail 设置") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "设置") || !strings.Contains(string(data), `segments = ["model"`) {
		t.Fatalf("config=%s", data)
	}
	cancelRoot := initProject(t)
	cancelPath := filepath.Join(cancelRoot, ".signalrail.toml")
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"config", "--lang", "en", "--scope", "project", "--project", cancelRoot}, strings.NewReader("2\nn\n"), &out, &stderr)
	if code != 0 || !strings.Contains(out.String(), "Preview") {
		t.Fatalf("cancel code=%d out=%q", code, out.String())
	}
	if _, err := os.Stat(cancelPath); !os.IsNotExist(err) {
		t.Fatalf("cancel wrote file: %v", err)
	}
}

func TestInstallDryRunAndStrictCodex(t *testing.T) {
	home := t.TempDir()
	root := initProject(t)
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"install", "claude", "--scope", "user", "--dry-run", "--home", home, "--project", root}, strings.NewReader(""), &out, &stderr)
	if code != 0 || !strings.Contains(out.String(), "statusLine") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "settings.json")); !os.IsNotExist(err) {
		t.Fatalf("dry-run wrote file: %v", err)
	}
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"install", "codex", "--scope", "project", "--strict", "--home", home, "--project", root, "--items", "model,cost"}, strings.NewReader(""), &out, &stderr)
	if code != 3 || !strings.Contains(stderr.String(), "cost") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(root, ".codex", "config.toml")); !os.IsNotExist(err) {
		t.Fatalf("strict wrote file: %v", err)
	}
}

func TestExplainAndDoctorJSON(t *testing.T) {
	root := initProject(t)
	home := t.TempDir()
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"explain", "--json", "--project", root, "--home", home}, strings.NewReader(""), &out, &stderr)
	var explain map[string]any
	if code != 0 || json.Unmarshal(out.Bytes(), &explain) != nil || len(explain["segments"].([]any)) != 6 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"doctor", "--json", "--project", root, "--home", home}, strings.NewReader(""), &out, &stderr)
	var doctor map[string]any
	if code != 1 || json.Unmarshal(out.Bytes(), &doctor) != nil || doctor["checks"] == nil {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
}

func TestInvalidLanguageAndVersion(t *testing.T) {
	var out, stderr bytes.Buffer
	if code := Run(context.Background(), []string{"config", "--lang", "fr"}, strings.NewReader(""), &out, &stderr); code != 2 {
		t.Fatalf("lang code=%d", code)
	}
	out.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"version"}, strings.NewReader(""), &out, &stderr); code != 0 || strings.TrimSpace(out.String()) == "" {
		t.Fatalf("version code=%d out=%q", code, out.String())
	}
}

func TestHelpAndRuntimeContract(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"help"}, {"render", "--help"}} {
		var out, stderr bytes.Buffer
		if code := Run(context.Background(), args, strings.NewReader(""), &out, &stderr); code != 0 || !strings.Contains(out.String(), "Usage:") || stderr.Len() != 0 {
			t.Fatalf("args=%v code=%d out=%q err=%q", args, code, out.String(), stderr.String())
		}
	}
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"render", "--runtime", "invalid"}, strings.NewReader("{}"), &out, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "auto, claude, or generic") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
}

func TestGenericInputBoundAndCommandSpecificValidation(t *testing.T) {
	root, home := initProject(t), t.TempDir()
	var out, stderr bytes.Buffer
	huge := `{"Model":{"Value":{"Name":"x"}},"padding":"` + strings.Repeat("x", adapter.MaxClaudeInputBytes) + `"}`
	code := Run(context.Background(), []string{"render", "--runtime", "generic", "--project", root, "--home", home}, strings.NewReader(huge), &out, &stderr)
	if code != 2 || out.Len() != 0 {
		t.Fatalf("oversized: code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"preview", "--scope", "user"}, strings.NewReader(""), &out, &stderr)
	if code != 2 || out.Len() != 0 {
		t.Fatalf("irrelevant option: code=%d out=%q", code, out.String())
	}
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"task", "show", "garbage", "--project", root}, strings.NewReader(""), &out, &stderr)
	if code != 2 {
		t.Fatalf("extra task argument code=%d", code)
	}
}

func TestRenderFormatsAndInvalidOptions(t *testing.T) {
	fixture, _ := os.ReadFile("../../testdata/claude/full.json")
	root, home := initProject(t), t.TempDir()
	for _, format := range []string{"plain", "ansi", "json"} {
		var out, stderr bytes.Buffer
		code := Run(context.Background(), []string{"render", "--runtime", "claude", "--format", format, "--project", root, "--home", home}, bytes.NewReader(fixture), &out, &stderr)
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("format=%s code=%d err=%q", format, code, stderr.String())
		}
		if format == "json" {
			var value map[string]any
			if json.Unmarshal(out.Bytes(), &value) != nil {
				t.Fatalf("invalid JSON: %s", out.String())
			}
		}
	}
	for _, args := range [][]string{{"render", "--runtime", "bogus"}, {"render", "--width", "0"}, {"render", "--format", "xml"}} {
		var out, stderr bytes.Buffer
		code := Run(context.Background(), args, bytes.NewReader(fixture), &out, &stderr)
		if code != 2 || out.Len() != 0 || stderr.Len() == 0 {
			t.Fatalf("args=%v code=%d out=%q err=%q", args, code, out.String(), stderr.String())
		}
	}
	var out, stderr bytes.Buffer
	generic := `{"Model":{}} {}`
	code := Run(context.Background(), []string{"render", "--runtime", "generic", "--project", root, "--home", home}, strings.NewReader(generic), &out, &stderr)
	if code != 2 {
		t.Fatalf("trailing generic code=%d out=%q", code, out.String())
	}
}

func TestConfigRejectsSymlinkAndSupportsUserScope(t *testing.T) {
	root, home := initProject(t), t.TempDir()
	target := filepath.Join(root, "target.toml")
	if err := os.WriteFile(target, []byte("version = 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, ".signalrail.toml")
	if err := os.Symlink(target, link); err == nil {
		var out, stderr bytes.Buffer
		code := Run(context.Background(), []string{"config", "--scope", "project", "--project", root, "--home", home}, strings.NewReader("2\ny\n"), &out, &stderr)
		if code != 2 {
			t.Fatalf("symlink code=%d out=%q err=%q", code, out.String(), stderr.String())
		}
	}
	if err := os.Remove(link); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"config", "--scope", "user", "--project", root, "--home", home}, strings.NewReader("1\ny\n"), &out, &stderr)
	if code != 0 {
		t.Fatalf("user code=%d err=%q", code, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(home, ".config", "signalrail", "config.toml")); err != nil {
		t.Fatal(err)
	}
}

func TestConfigEnglishDefaultPreviewAndInvalidSelection(t *testing.T) {
	root := initProject(t)
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"config", "--lang", "en", "--scope", "project", "--project", root}, strings.NewReader("\ny\n"), &out, &stderr)
	if code != 0 || !strings.Contains(out.String(), "Preview") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	if _, err := config.Load(root, t.TempDir(), config.RuntimeClaude, config.Overrides{}); err != nil {
		t.Fatalf("saved TOML invalid: %v", err)
	}
	other := initProject(t)
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"config", "--scope", "project", "--project", other}, strings.NewReader("9\n"), &out, &stderr)
	if code != 2 {
		t.Fatalf("invalid selection code=%d", code)
	}
}

func TestTaskBlockAndInstallApplyPreserveUnrelatedSettings(t *testing.T) {
	root, home := initProject(t), t.TempDir()
	var out, stderr bytes.Buffer
	if code := Run(context.Background(), []string{"task", "set", "Ship", "--project", root}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatal(stderr.String())
	}
	out.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"task", "block", "waiting", "--project", root}, strings.NewReader(""), &out, &stderr); code != 0 || !strings.Contains(out.String(), "waiting") {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	settings := filepath.Join(home, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(settings), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	if code := Run(context.Background(), []string{"install", "claude", "--home", home, "--project", root}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatalf("code=%d err=%q", code, stderr.String())
	}
	data, _ := os.ReadFile(settings)
	if !strings.Contains(string(data), `"theme": "dark"`) || !strings.Contains(string(data), "signalrail render") {
		t.Fatalf("settings=%s", data)
	}
	backups, _ := filepath.Glob(settings + ".bak.*")
	if len(backups) != 1 {
		t.Fatalf("backups=%v", backups)
	}
}

func TestDoctorValidatesRuntimeConfigurationAndExplainTextHasAge(t *testing.T) {
	root, home := initProject(t), t.TempDir()
	var out, stderr bytes.Buffer
	claude, codex := filepath.Join(home, ".claude", "settings.json"), filepath.Join(home, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(claude), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(codex), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claude, []byte(`{"statusLine":{"type":"command","command":"signalrail render --runtime claude"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte("[tui]\n# SignalRail managed status line\nstatus_line = [\"model-with-reasoning\", \"context-remaining\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	code := Run(context.Background(), []string{"doctor", "--json", "--home", home, "--project", root}, strings.NewReader(""), &out, &stderr)
	if code != 0 {
		t.Fatalf("healthy code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	if err := os.WriteFile(claude, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	if code = Run(context.Background(), []string{"doctor", "--json", "--home", home, "--project", root}, strings.NewReader(""), &out, &stderr); code != 2 {
		t.Fatalf("invalid code=%d out=%q", code, out.String())
	}
	if err := os.Remove(claude); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	if code = Run(context.Background(), []string{"explain", "--home", home, "--project", root}, strings.NewReader(""), &out, &stderr); code != 0 || !strings.Contains(out.String(), "age_ms=") {
		t.Fatalf("explain code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
}

func TestRenderRequiredWidthsEndToEnd(t *testing.T) {
	fixture, _ := os.ReadFile("../../testdata/claude/full.json")
	root, home := initProject(t), t.TempDir()
	for _, width := range []int{40, 60, 80, 120, 160} {
		var out, stderr bytes.Buffer
		code := Run(context.Background(), []string{"render", "--runtime", "claude", "--width", strconv.Itoa(width), "--project", root, "--home", home}, bytes.NewReader(fixture), &out, &stderr)
		line := strings.TrimSuffix(out.String(), "\n")
		if code != 0 || strings.Contains(line, "\n") || statusrender.VisibleWidth(line) > width {
			t.Fatalf("width=%d code=%d visible=%d out=%q err=%q", width, code, statusrender.VisibleWidth(line), out.String(), stderr.String())
		}
	}
}

func TestDoctorUsesProjectOverrideAndRequiresSignalRailMarker(t *testing.T) {
	root, home := initProject(t), t.TempDir()
	userClaude := filepath.Join(home, ".claude", "settings.json")
	userCodex := filepath.Join(home, ".codex", "config.toml")
	projectClaude := filepath.Join(root, ".claude", "settings.json")
	projectCodex := filepath.Join(root, ".codex", "config.toml")
	for _, path := range []string{userClaude, userCodex, projectClaude, projectCodex} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(userClaude, []byte("{"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(userCodex, []byte("[tui\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectClaude, []byte(`{"statusLine":{"type":"command","command":"signalrail render"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectCodex, []byte("[tui]\n# SignalRail managed status line\nstatus_line=[\"model-with-reasoning\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"doctor", "--json", "--home", home, "--project", root}, strings.NewReader(""), &out, &stderr)
	if code != 0 {
		t.Fatalf("project override code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
	if err := os.WriteFile(projectCodex, []byte("[tui]\nstatus_line=[\"model-with-reasoning\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	stderr.Reset()
	code = Run(context.Background(), []string{"doctor", "--json", "--home", home, "--project", root}, strings.NewReader(""), &out, &stderr)
	if code != 1 {
		t.Fatalf("unmarked code=%d out=%q", code, out.String())
	}
}

func TestDoctorAcceptsManagedCodexRailWithoutModel(t *testing.T) {
	root, home := initProject(t), t.TempDir()
	claude := filepath.Join(root, ".claude", "settings.json")
	codex := filepath.Join(root, ".codex", "config.toml")
	if err := os.MkdirAll(filepath.Dir(claude), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(codex), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(claude, []byte(`{"statusLine":{"type":"command","command":"signalrail render"}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(codex, []byte("[tui]\n# SignalRail managed status line\nstatus_line=[\"project-name\", \"context-remaining\"]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	if code := Run(context.Background(), []string{"doctor", "--json", "--home", home, "--project", root}, strings.NewReader(""), &out, &stderr); code != 0 {
		t.Fatalf("code=%d out=%q err=%q", code, out.String(), stderr.String())
	}
}

func TestOptionValueCannotConsumeAnotherOption(t *testing.T) {
	var out, stderr bytes.Buffer
	code := Run(context.Background(), []string{"render", "--width", "--format", "plain"}, strings.NewReader("{}"), &out, &stderr)
	if code != 2 || !strings.Contains(stderr.String(), "--width requires a value") {
		t.Fatalf("code=%d err=%q", code, stderr.String())
	}
}

func initProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	return root
}
