package install

import (
	"fmt"
	"strconv"
	"strings"

	toml "github.com/pelletier/go-toml/v2"
)

var pinnedCodexItems = map[string]bool{
	"model": true, "model-with-reasoning": true, "reasoning": true, "current-dir": true,
	"project-name": true, "git-branch": true, "run-state": true, "context-remaining": true,
	"context-used": true, "five-hour-limit": true, "weekly-limit": true, "codex-version": true,
	"context-window-size": true, "used-tokens": true, "thread-id": true, "fast-mode": true,
	"thread-title": true, "task-progress": true,
}

var pinnedTerminalTitleItems = map[string]bool{
	"app-name": true, "project-name": true, "current-dir": true, "activity": true,
	"run-state": true, "thread-title": true, "git-branch": true, "context-remaining": true,
	"context-used": true, "five-hour-limit": true, "weekly-limit": true, "codex-version": true,
	"used-tokens": true, "total-input-tokens": true, "total-output-tokens": true,
	"thread-id": true, "fast-mode": true, "model": true, "model-with-reasoning": true,
	"reasoning": true, "task-progress": true,
}

var intentMap = map[string][]string{
	"model": {"model-with-reasoning"}, "project": {"project-name"}, "branch": {"git-branch"},
	"progress": {"task-progress"}, "context": {"context-remaining"},
	"quota": {"five-hour-limit", "weekly-limit"},
}

func PlanCodex(path string, items, title []string) (Change, []Warning, error) {
	before, err := readOptional(path)
	if err != nil {
		return Change{}, nil, fmt.Errorf("read Codex config: %w", err)
	}
	return PlanCodexBytes(path, before, items, title)
}

func PlanCodexBytes(path string, before []byte, items, title []string) (Change, []Warning, error) {
	if len(before) > 0 {
		var check map[string]any
		if err := toml.Unmarshal(before, &check); err != nil {
			return Change{}, nil, fmt.Errorf("decode Codex config: %w", err)
		}
	}
	mapped, warnings := mapCodexItems(items)
	mappedTitle, titleWarnings := mapTerminalTitleItems(title)
	warnings = append(warnings, titleWarnings...)
	owned := []string{"# SignalRail managed status line", "status_line = " + tomlArray(mapped), "status_line_use_colors = true"}
	if len(mappedTitle) > 0 {
		owned = append(owned, "terminal_title = "+tomlArray(mappedTitle))
	}
	after := patchTUI(string(before), owned, len(mappedTitle) > 0)
	return Change{Path: path, Before: append([]byte(nil), before...), After: []byte(after)}, warnings, nil
}

func mapCodexItems(items []string) ([]string, []Warning) {
	var mapped []string
	var warnings []Warning
	seen := map[string]bool{}
	for _, item := range items {
		values, ok := intentMap[item]
		if !ok && pinnedCodexItems[item] {
			values, ok = []string{item}, true
		}
		if !ok {
			warnings = append(warnings, unsupportedItemWarning(item))
			continue
		}
		for _, value := range values {
			if !seen[value] {
				seen[value] = true
				mapped = append(mapped, value)
			}
		}
	}
	return mapped, warnings
}

func unsupportedItemWarning(item string) Warning {
	codes := map[string]string{
		"cost": "codex-cost-unsupported", "forecast": "codex-forecast-unsupported",
		"task": "codex-task-state-unsupported", "custom-task-text": "codex-custom-task-unsupported",
	}
	code := codes[item]
	if code == "" {
		code = "unsupported-codex-item"
	}
	return Warning{Code: code, Message: fmt.Sprintf("Codex cannot render %q from SignalRail", item)}
}

func mapTerminalTitleItems(items []string) ([]string, []Warning) {
	aliases := map[string]string{"project": "project-name", "spinner": "activity", "status": "run-state", "thread": "thread-title"}
	var mapped []string
	var warnings []Warning
	seen := map[string]bool{}
	for _, item := range items {
		if alias := aliases[item]; alias != "" {
			item = alias
		}
		if !pinnedTerminalTitleItems[item] {
			warnings = append(warnings, Warning{Code: "unsupported-codex-title", Message: fmt.Sprintf("Codex cannot use %q in terminal title", item)})
			continue
		}
		if !seen[item] {
			seen[item] = true
			mapped = append(mapped, item)
		}
	}
	return mapped, warnings
}

func patchTUI(source string, owned []string, manageTerminalTitle bool) string {
	trimmed := strings.TrimSuffix(strings.ReplaceAll(source, "\r\n", "\n"), "\n")
	lines := []string{}
	if trimmed != "" {
		lines = strings.Split(trimmed, "\n")
	}
	start, end := -1, len(lines)
	for i, line := range lines {
		section := strings.TrimSpace(line)
		if section == "[tui]" {
			start = i
			continue
		}
		if start >= 0 && i > start && strings.HasPrefix(section, "[") {
			end = i
			break
		}
	}
	if start < 0 {
		if len(lines) > 0 && lines[len(lines)-1] != "" {
			lines = append(lines, "")
		}
		lines = append(lines, "[tui]")
		lines = append(lines, owned...)
		return strings.Join(lines, "\n") + "\n"
	}
	result := append([]string(nil), lines[:start+1]...)
	result = append(result, owned...)
	for _, line := range lines[start+1 : end] {
		if !ownedKey(line, manageTerminalTitle) {
			result = append(result, line)
		}
	}
	result = append(result, lines[end:]...)
	return strings.Join(result, "\n") + "\n"
}

func ownedKey(line string, manageTerminalTitle bool) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "# SignalRail managed status line" {
		return true
	}
	keys := []string{"status_line", "status_line_use_colors"}
	if manageTerminalTitle {
		keys = append(keys, "terminal_title")
	}
	for _, key := range keys {
		if strings.HasPrefix(trimmed, key+" ") || strings.HasPrefix(trimmed, key+"=") {
			return true
		}
	}
	return false
}

func tomlArray(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return "[" + strings.Join(quoted, ", ") + "]"
}
