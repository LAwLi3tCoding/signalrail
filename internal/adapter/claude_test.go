package adapter

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/LAwLi3tCoding/signalrail/internal/status"
)

func TestParseClaudeFullPayload(t *testing.T) {
	f, err := os.Open("../../testdata/claude/full.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	now := time.Date(2026, 6, 19, 1, 2, 3, 0, time.UTC)
	got, err := ParseClaude(f, now)
	if err != nil {
		t.Fatal(err)
	}
	if got.Model.Value.Name != "Opus 4.7" || got.Model.Value.Effort != "high" {
		t.Fatalf("model=%+v", got.Model)
	}
	if got.Model.Confidence != status.ConfidenceExact || got.Model.Freshness != status.FreshnessFresh {
		t.Fatalf("model provenance=%+v", got.Model)
	}
	if got.Project.Value.Name != "signalrail" {
		t.Fatalf("project=%+v", got.Project)
	}
	if got.Context.Value.UsedPercent != 62 || got.Context.Value.LeftLabel != "38%" {
		t.Fatalf("context=%+v", got.Context)
	}
	if got.Cost.Confidence != status.ConfidenceEstimated || got.Cost.Value.Amount != 1.24 {
		t.Fatalf("cost=%+v", got.Cost)
	}
	if got.Model.ObservedAt != now {
		t.Fatalf("observed=%v", got.Model.ObservedAt)
	}
}

func TestParseClaudePartialMarksMissingDataUnavailable(t *testing.T) {
	f, err := os.Open("../../testdata/claude/partial.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	got, err := ParseClaude(f, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got.Cost.Confidence != status.ConfidenceUnavailable || got.Context.Confidence != status.ConfidenceUnavailable {
		t.Fatalf("snapshot=%+v", got)
	}
}

func TestParseClaudeDoesNotInventMissingProject(t *testing.T) {
	got, err := ParseClaude(strings.NewReader(`{"model":{"display_name":"Opus"}}`), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got.Project.Confidence != status.ConfidenceUnavailable || got.Project.Value.Name != "" {
		t.Fatalf("project=%+v", got.Project)
	}
}

func TestParseClaudeRejectsMalformedTrailingAndOversizedInput(t *testing.T) {
	for name, input := range map[string]string{
		"malformed": "{",
		"trailing":  `{"model":{}} {}`,
		"oversized": `{"padding":"` + strings.Repeat("x", MaxClaudeInputBytes) + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := ParseClaude(strings.NewReader(input), time.Now()); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseClaudeRejectsTopLevelNullAndMalformedFixture(t *testing.T) {
	for _, name := range []string{"null.json", "malformed.json"} {
		f, err := os.Open("../../testdata/claude/" + name)
		if err != nil {
			t.Fatal(err)
		}
		_, parseErr := ParseClaude(f, time.Now())
		f.Close()
		if parseErr == nil {
			t.Fatalf("%s: expected error", name)
		}
	}
}

func TestParseClaudePreservesDecimalContextAndFutureFields(t *testing.T) {
	input := `{"model":{"display_name":"Opus"},"workspace":{"current_dir":"/tmp/demo"},"context_window":{"used_percentage":62.5,"remaining_percentage":37.5}}`
	got, err := ParseClaude(strings.NewReader(input), time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if got.Context.Value.UsedPercent != 62.5 || got.Context.Value.LeftLabel != "37.5%" {
		t.Fatalf("context=%+v", got.Context)
	}
	f, err := os.Open("../../testdata/claude/future.json")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := ParseClaude(f, time.Now()); err != nil {
		t.Fatalf("future fields rejected: %v", err)
	}
}
