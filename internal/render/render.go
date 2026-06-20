package render

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/LAwLi3t-CN/signalrail/internal/config"
	"github.com/LAwLi3t-CN/signalrail/internal/status"
)

type Options struct {
	Width    int
	Format   string
	Color    bool
	ASCII    bool
	Now      time.Time
	HomeDir  string
	UserName string
}

type Result struct {
	Output    string               `json:"output"`
	Included  []status.SegmentName `json:"included"`
	Shortened []status.SegmentName `json:"shortened"`
	Omitted   []status.SegmentName `json:"omitted"`
}

type profile int

const (
	full profile = iota
	standard
	compact
	minimal
)

type candidate struct {
	name     status.SegmentName
	variants [4]string
	color    int
}

type selected struct {
	name      status.SegmentName
	text      string
	color     int
	shortened bool
}

func Render(snapshot status.Snapshot, cfg config.Config, opts Options) (Result, error) {
	if opts.Width <= 0 {
		opts.Width = 120
	}
	if opts.Format == "" {
		opts.Format = "plain"
	}
	if opts.Format != "plain" && opts.Format != "ansi" {
		return Result{}, fmt.Errorf("unsupported render format %q", opts.Format)
	}
	candidates := buildCandidates(snapshot, cfg, opts)
	start := profileFor(opts.Width)
	for mode := start; mode <= minimal; mode++ {
		parts := selectCandidates(cfg.Segments, candidates, mode)
		if VisibleWidth(joinPlain(parts)) <= opts.Width {
			return resultFor(parts, cfg, opts), nil
		}
	}
	parts := selectCandidates(cfg.Segments, candidates, minimal)
	parts = squeeze(parts, opts.Width, opts.ASCII)
	return resultFor(parts, cfg, opts), nil
}

func profileFor(width int) profile {
	switch {
	case width >= 160:
		return full
	case width >= 100:
		return standard
	case width >= 60:
		return compact
	default:
		return minimal
	}
}

func buildCandidates(snapshot status.Snapshot, cfg config.Config, opts Options) map[status.SegmentName]candidate {
	out := map[status.SegmentName]candidate{}
	if model := snapshot.Model.Value; snapshot.Model.Confidence != status.ConfidenceUnavailable && model.Name != "" {
		name, effort := clean(model.Name, cfg, opts), clean(model.Effort, cfg, opts)
		mark := marker(snapshot.Model.Confidence, snapshot.Model.Freshness, opts.ASCII)
		fullText := glyph("◆ ", "MODEL ", opts.ASCII) + name
		standardText := glyph("◆ ", "", opts.ASCII) + name
		if effort != "" {
			fullText += glyph(" · ", " ", opts.ASCII) + effort
			standardText += " " + effort
		}
		out[status.SegmentModel] = candidate{status.SegmentModel, [4]string{fullText + mark, standardText + mark, name + mark, name + mark}, 36}
	}
	if project := snapshot.Project.Value; snapshot.Project.Confidence != status.ConfidenceUnavailable && project.Name != "" {
		name := clean(project.Name, cfg, opts)
		branch := clean(sanitizeBranch(project.Branch, cfg), cfg, opts)
		fullName := name
		if branch != "" {
			fullName += "/" + branch
		}
		if project.Dirty {
			fullName += "*"
		}
		mark := marker(snapshot.Project.Confidence, snapshot.Project.Freshness, opts.ASCII)
		out[status.SegmentProject] = candidate{status.SegmentProject, [4]string{glyph("◇ ", "PROJECT ", opts.ASCII) + fullName + mark, glyph("◇ ", "", opts.ASCII) + fullName + mark, name + mark, ""}, 34}
	}
	if task := snapshot.Task.Value; snapshot.Task.Confidence != status.ConfidenceUnavailable && task.Title != "" {
		title := clean(task.Title, cfg, opts)
		prefix := glyph("▶ ", "TASK ", opts.ASCII)
		if task.State == "blocked" {
			prefix = "! "
		} else if task.State == "done" {
			prefix = glyph("✓ ", "[x] ", opts.ASCII)
		}
		mark := marker(snapshot.Task.Confidence, snapshot.Task.Freshness, opts.ASCII)
		out[status.SegmentTask] = candidate{status.SegmentTask, [4]string{prefix + title + mark, prefix + title + mark, title + mark, title + mark}, 33}
		if task.TotalSteps > 0 {
			ratio := fmt.Sprintf("%d/%d", task.Step, task.TotalSteps)
			out[status.SegmentProgress] = candidate{status.SegmentProgress, [4]string{ratio + " " + progressBar(task.Step, task.TotalSteps, opts.ASCII), ratio, ratio, ""}, 32}
		}
	}
	if context := snapshot.Context.Value; snapshot.Context.Confidence != status.ConfidenceUnavailable && context.LeftLabel != "" {
		left := clean(context.LeftLabel, cfg, opts)
		mark := marker(snapshot.Context.Confidence, snapshot.Context.Freshness, opts.ASCII)
		color := 33
		if context.UsedPercent >= 80 {
			color = 31
		}
		out[status.SegmentContext] = candidate{status.SegmentContext, [4]string{"CTX " + left + " left" + mark, "CTX " + left + " left" + mark, "C" + left + mark, "C" + left + mark}, color}
	}
	if cfg.Cost.Enabled && snapshot.Cost.Confidence != status.ConfidenceUnavailable {
		currency := "$"
		if snapshot.Cost.Value.Currency != "" && snapshot.Cost.Value.Currency != "USD" {
			currency = clean(snapshot.Cost.Value.Currency, cfg, opts) + " "
		}
		value := fmt.Sprintf("%s%.2f%s", currency, snapshot.Cost.Value.Amount, marker(snapshot.Cost.Confidence, snapshot.Cost.Freshness, opts.ASCII))
		out[status.SegmentCost] = candidate{status.SegmentCost, [4]string{value, "", "", ""}, 35}
	}
	return out
}

func selectCandidates(order []status.SegmentName, candidates map[status.SegmentName]candidate, mode profile) []selected {
	parts := make([]selected, 0, len(order))
	for _, name := range order {
		item, ok := candidates[name]
		if !ok {
			continue
		}
		text := item.variants[int(mode)]
		if text == "" {
			continue
		}
		parts = append(parts, selected{name: name, text: text, color: item.color, shortened: mode != full})
	}
	return parts
}

func squeeze(parts []selected, width int, ascii bool) []selected {
	parts = append([]selected(nil), parts...)
	for VisibleWidth(joinPlain(parts)) > width && len(parts) > 0 {
		over := VisibleWidth(joinPlain(parts)) - width
		index := shrinkIndex(parts)
		current := VisibleWidth(parts[index].text)
		target := current - over
		if target < 1 {
			target = 1
		}
		parts[index].text = truncateTail(parts[index].text, target, ascii)
		parts[index].shortened = true
		if current == VisibleWidth(parts[index].text) || (target == 1 && VisibleWidth(joinPlain(parts)) > width) {
			parts = append(parts[:index], parts[index+1:]...)
		}
	}
	return parts
}

func shrinkIndex(parts []selected) int {
	priority := []status.SegmentName{status.SegmentCost, status.SegmentProgress, status.SegmentProject, status.SegmentContext, status.SegmentTask, status.SegmentModel}
	for _, wanted := range priority {
		for i, part := range parts {
			if part.name == wanted {
				return i
			}
		}
	}
	return len(parts) - 1
}

func resultFor(parts []selected, cfg config.Config, opts Options) Result {
	result := Result{}
	painted := make([]string, 0, len(parts))
	for _, part := range parts {
		painted = append(painted, paint(part.text, part.color, opts))
		result.Included = append(result.Included, part.name)
		if part.shortened {
			result.Shortened = append(result.Shortened, part.name)
		}
	}
	for _, name := range cfg.Segments {
		if !contains(result.Included, name) {
			result.Omitted = append(result.Omitted, name)
		}
	}
	result.Output = strings.Join(painted, "  ")
	return result
}

func joinPlain(parts []selected) string {
	values := make([]string, len(parts))
	for i, part := range parts {
		values[i] = part.text
	}
	return strings.Join(values, "  ")
}

func marker(confidence status.Confidence, freshness status.Freshness, ascii bool) string {
	if freshness == status.FreshnessStale || freshness == status.FreshnessDegraded {
		return "!"
	}
	if freshness == status.FreshnessCached {
		return glyph("↻", "^", ascii)
	}
	if confidence == status.ConfidenceEstimated {
		return "~"
	}
	return ""
}

func progressBar(step, total int, ascii bool) string {
	const width = 7
	filled := 0
	if total > 0 {
		filled = (step*width + total/2) / total
	}
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	if ascii {
		return strings.Repeat("#", filled) + strings.Repeat("-", width-filled)
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

func clean(value string, cfg config.Config, opts Options) string {
	value = StripANSI(value)
	value = strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, value)
	value = strings.Join(strings.Fields(value), " ")
	if cfg.Privacy.RedactPaths && opts.HomeDir != "" {
		home := filepath.Clean(opts.HomeDir)
		for _, variant := range []string{opts.HomeDir, home, strings.ReplaceAll(home, "\\", "/"), strings.ReplaceAll(home, "/", "\\")} {
			if variant != "" {
				value = strings.ReplaceAll(value, variant, "~")
			}
		}
	}
	if cfg.Privacy.RedactUser && opts.UserName != "" {
		value = strings.ReplaceAll(value, opts.UserName, "[user]")
	}
	return value
}

func sanitizeBranch(branch string, cfg config.Config) string {
	for _, pattern := range cfg.Privacy.SensitiveBranches {
		if matched, _ := filepath.Match(pattern, branch); matched {
			return "[redacted]"
		}
	}
	return branch
}
func glyph(unicodeValue, asciiValue string, ascii bool) string {
	if ascii {
		return asciiValue
	}
	return unicodeValue
}
func paint(text string, color int, opts Options) string {
	if opts.Format != "ansi" || !opts.Color || text == "" {
		return text
	}
	return fmt.Sprintf("\x1b[%dm%s\x1b[0m", color, text)
}
func contains(items []status.SegmentName, wanted status.SegmentName) bool {
	for _, item := range items {
		if item == wanted {
			return true
		}
	}
	return false
}
