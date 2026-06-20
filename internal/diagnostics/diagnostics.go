package diagnostics

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/LAwLi3t-CN/signalrail/internal/config"
	"github.com/LAwLi3t-CN/signalrail/internal/render"
	"github.com/LAwLi3t-CN/signalrail/internal/status"
	toml "github.com/pelletier/go-toml/v2"
)

type Check struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Message string `json:"message"`
}
type Report struct {
	Status string  `json:"status"`
	Checks []Check `json:"checks"`
}
type Segment struct {
	Name       status.SegmentName `json:"name"`
	Included   bool               `json:"included"`
	Confidence status.Confidence  `json:"confidence"`
	Freshness  status.Freshness   `json:"freshness"`
	Source     status.Source      `json:"source"`
	AgeMS      int64              `json:"age_ms"`
}
type Explanation struct {
	Output   string    `json:"output"`
	Segments []Segment `json:"segments"`
}

func Doctor(home, project string) (Report, int) {
	report := Report{Status: "ok"}
	for _, runtime := range []config.Runtime{config.RuntimeClaude, config.RuntimeCodex} {
		if _, err := config.Load(project, home, runtime, config.Overrides{}); err != nil {
			report.Status = "error"
			report.Checks = append(report.Checks, Check{string(runtime) + "-config", "error", err.Error()})
			return report, 2
		}
		report.Checks = append(report.Checks, Check{string(runtime) + "-config", "ok", "SignalRail configuration is valid"})
	}
	claudeCheck, claudeCode := checkClaude([]string{filepath.Join(project, ".claude", "settings.json"), filepath.Join(home, ".claude", "settings.json")})
	codexCheck, codexCode := checkCodex([]string{filepath.Join(project, ".codex", "config.toml"), filepath.Join(home, ".codex", "config.toml")})
	report.Checks = append(report.Checks, claudeCheck, codexCheck, Check{"codex-capability", "ok", "native fields only; external rendering is not claimed"})
	if claudeCode == 2 || codexCode == 2 {
		report.Status = "error"
		return report, 2
	}
	if claudeCode == 1 || codexCode == 1 {
		report.Status = "warning"
		return report, 1
	}
	return report, 0
}

func Explain(snapshot status.Snapshot, result render.Result, _ []status.SegmentName, now time.Time) Explanation {
	explanation := Explanation{Output: result.Output}
	for _, name := range []status.SegmentName{status.SegmentModel, status.SegmentProject, status.SegmentTask, status.SegmentProgress, status.SegmentContext, status.SegmentCost} {
		confidence, freshness, source, observed := metadata(snapshot, name)
		age := int64(0)
		if !observed.IsZero() && now.After(observed) {
			age = now.Sub(observed).Milliseconds()
		}
		explanation.Segments = append(explanation.Segments, Segment{Name: name, Included: has(result.Included, name), Confidence: confidence, Freshness: freshness, Source: source, AgeMS: age})
	}
	return explanation
}

func checkClaude(paths []string) (Check, int) {
	path := firstExisting(paths)
	if path == "" {
		return Check{"claude-install", "warning", "SignalRail statusLine is not installed"}, 1
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Check{"claude-install", "error", err.Error()}, 2
	}
	var root map[string]any
	if err := json.Unmarshal(data, &root); err != nil {
		return Check{"claude-install", "error", fmt.Sprintf("invalid Claude settings: %v", err)}, 2
	}
	line, ok := root["statusLine"].(map[string]any)
	command, _ := line["command"].(string)
	if !ok || line["type"] != "command" || !strings.Contains(command, "signalrail") {
		return Check{"claude-install", "warning", "Claude statusLine does not invoke SignalRail"}, 1
	}
	return Check{"claude-install", "ok", "Claude statusLine invokes SignalRail"}, 0
}

func checkCodex(paths []string) (Check, int) {
	path := firstExisting(paths)
	if path == "" {
		return Check{"codex-install", "warning", "SignalRail Codex status line is not installed"}, 1
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Check{"codex-install", "error", err.Error()}, 2
	}
	var document struct {
		TUI struct {
			StatusLine []string `toml:"status_line"`
		} `toml:"tui"`
	}
	if err := toml.Unmarshal(data, &document); err != nil {
		return Check{"codex-install", "error", fmt.Sprintf("invalid Codex config: %v", err)}, 2
	}
	if len(document.TUI.StatusLine) == 0 || !strings.Contains(string(data), "# SignalRail managed status line") {
		return Check{"codex-install", "warning", "Codex status_line is not managed by SignalRail"}, 1
	}
	return Check{"codex-install", "ok", "Codex native status_line is configured"}, 0
}

func metadata(snapshot status.Snapshot, name status.SegmentName) (status.Confidence, status.Freshness, status.Source, time.Time) {
	switch name {
	case status.SegmentModel:
		return snapshot.Model.Confidence, snapshot.Model.Freshness, snapshot.Model.Source, snapshot.Model.ObservedAt
	case status.SegmentProject:
		return snapshot.Project.Confidence, snapshot.Project.Freshness, snapshot.Project.Source, snapshot.Project.ObservedAt
	case status.SegmentTask, status.SegmentProgress:
		return snapshot.Task.Confidence, snapshot.Task.Freshness, snapshot.Task.Source, snapshot.Task.ObservedAt
	case status.SegmentContext:
		return snapshot.Context.Confidence, snapshot.Context.Freshness, snapshot.Context.Source, snapshot.Context.ObservedAt
	case status.SegmentCost:
		return snapshot.Cost.Confidence, snapshot.Cost.Freshness, snapshot.Cost.Source, snapshot.Cost.ObservedAt
	default:
		return status.ConfidenceUnavailable, status.FreshnessDegraded, status.Source{}, time.Time{}
	}
}
func firstExisting(paths []string) string {
	for _, path := range paths {
		if exists(path) {
			return path
		}
	}
	return ""
}
func exists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular()
}
func has(items []status.SegmentName, wanted status.SegmentName) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}
