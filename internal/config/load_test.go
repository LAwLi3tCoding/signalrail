package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/LAwLi3t-CN/signalrail/internal/status"
)

func TestLoadDefaultsForCodex(t *testing.T) {
	projectDir := t.TempDir()
	userDir := t.TempDir()

	cfg, err := Load(projectDir, userDir, RuntimeCodex, Overrides{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Version != CurrentVersion {
		t.Fatalf("Version = %d, want %d", cfg.Version, CurrentVersion)
	}

	wantSegments := []status.SegmentName{
		status.SegmentModel,
		status.SegmentProject,
		status.SegmentProgress,
		status.SegmentContext,
	}
	if !reflect.DeepEqual(cfg.Segments, wantSegments) {
		t.Fatalf("Segments = %#v, want %#v", cfg.Segments, wantSegments)
	}

	if !cfg.Privacy.RedactUser || !cfg.Privacy.RedactPaths {
		t.Fatalf("Privacy = %#v, want redact flags enabled", cfg.Privacy)
	}

	if cfg.Cost.Enabled {
		t.Fatalf("Cost.Enabled = true, want false")
	}
}

func TestLoadAppliesPrecedence(t *testing.T) {
	projectDir := t.TempDir()
	userDir := t.TempDir()

	writeConfig(t, filepath.Join(userDir, ".config", "signalrail", "config.toml"), `
segments = ["model", "project", "cost"]

[privacy]
redact_paths = false

[task]
default_label = "User general"

[runtime.claude.context]
label = "remaining"
`)

	writeConfig(t, filepath.Join(projectDir, ".signalrail.toml"), `
[task]
default_label = "Project general"

[cost]
enabled = true

[runtime.claude]
segments = ["model", "project", "task"]

[runtime.claude.task]
default_label = "Project runtime"

[runtime.claude.privacy]
sensitive_branches = ["release/*"]
`)

	cfg, err := Load(projectDir, userDir, RuntimeClaude, Overrides{
		Task: TaskOverrides{
			DefaultLabel: ptr("CLI"),
		},
		Cost: CostOverrides{
			Currency: ptr("EUR"),
		},
	})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	wantSegments := []status.SegmentName{
		status.SegmentModel,
		status.SegmentProject,
		status.SegmentTask,
	}
	if !reflect.DeepEqual(cfg.Segments, wantSegments) {
		t.Fatalf("Segments = %#v, want %#v", cfg.Segments, wantSegments)
	}

	if cfg.Task.DefaultLabel != "CLI" {
		t.Fatalf("Task.DefaultLabel = %q, want %q", cfg.Task.DefaultLabel, "CLI")
	}

	if cfg.Context.Label != "remaining" {
		t.Fatalf("Context.Label = %q, want %q", cfg.Context.Label, "remaining")
	}

	if cfg.Cost.Enabled != true {
		t.Fatalf("Cost.Enabled = %v, want true", cfg.Cost.Enabled)
	}

	if cfg.Cost.Currency != "EUR" {
		t.Fatalf("Cost.Currency = %q, want %q", cfg.Cost.Currency, "EUR")
	}

	if cfg.Privacy.RedactPaths != false {
		t.Fatalf("Privacy.RedactPaths = %v, want false", cfg.Privacy.RedactPaths)
	}

	wantSensitiveBranches := []string{"release/*"}
	if !reflect.DeepEqual(cfg.Privacy.SensitiveBranches, wantSensitiveBranches) {
		t.Fatalf("Privacy.SensitiveBranches = %#v, want %#v", cfg.Privacy.SensitiveBranches, wantSensitiveBranches)
	}
}

func TestLoadReplacesArraysInsteadOfMerging(t *testing.T) {
	projectDir := t.TempDir()
	userDir := t.TempDir()

	writeConfig(t, filepath.Join(userDir, ".config", "signalrail", "config.toml"), `
segments = ["model", "project", "task", "cost"]
`)

	writeConfig(t, filepath.Join(projectDir, ".signalrail.toml"), `
[runtime.claude]
segments = ["model", "context"]
`)

	cfg, err := Load(projectDir, userDir, RuntimeClaude, Overrides{})
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	wantSegments := []status.SegmentName{
		status.SegmentModel,
		status.SegmentContext,
	}
	if !reflect.DeepEqual(cfg.Segments, wantSegments) {
		t.Fatalf("Segments = %#v, want replacement %#v", cfg.Segments, wantSegments)
	}
}

func TestLoadUsesXDGConfigHomeForCurrentUser(t *testing.T) {
	home, xdg, project := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", xdg)
	writeConfig(t, filepath.Join(xdg, "signalrail", "config.toml"), `
segments = ["model", "context"]
`)

	cfg, err := Load(project, home, RuntimeClaude, Overrides{})
	if err != nil {
		t.Fatal(err)
	}
	want := []status.SegmentName{status.SegmentModel, status.SegmentContext}
	if !reflect.DeepEqual(cfg.Segments, want) {
		t.Fatalf("Segments = %#v, want XDG config %#v", cfg.Segments, want)
	}
	if got := UserPath(home); got != filepath.Join(xdg, "signalrail", "config.toml") {
		t.Fatalf("UserPath() = %q", got)
	}
}

func TestLoadRejectsUnknownKeys(t *testing.T) {
	projectDir := t.TempDir()
	userDir := t.TempDir()

	writeConfig(t, filepath.Join(projectDir, ".signalrail.toml"), `
[privacy]
redact_user = true
unknown_toggle = true
`)

	_, err := Load(projectDir, userDir, RuntimeClaude, Overrides{})
	if err == nil {
		t.Fatal("Load() error = nil, want unknown-key failure")
	}

	if !strings.Contains(strings.ToLower(err.Error()), "unknown") {
		t.Fatalf("Load() error = %q, want mention of unknown key", err)
	}
}

func writeConfig(t *testing.T, path, contents string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(contents, "\n")), 0o644); err != nil {
		t.Fatalf("WriteFile(%q) error = %v", path, err)
	}
}

func ptr[T any](value T) *T {
	return &value
}
