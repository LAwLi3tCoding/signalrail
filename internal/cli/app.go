package cli

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/LAwLi3t-CN/signalrail/internal/adapter"
	"github.com/LAwLi3t-CN/signalrail/internal/config"
	"github.com/LAwLi3t-CN/signalrail/internal/diagnostics"
	"github.com/LAwLi3t-CN/signalrail/internal/install"
	"github.com/LAwLi3t-CN/signalrail/internal/provider"
	"github.com/LAwLi3t-CN/signalrail/internal/render"
	"github.com/LAwLi3t-CN/signalrail/internal/status"
	taskstate "github.com/LAwLi3t-CN/signalrail/internal/task"
)

const Version = "0.1.0-dev"

func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"config"}
	}
	if args[0] == "--help" || args[0] == "-h" {
		writeHelp(stdout, "")
		return 0
	}
	if args[0] == "help" {
		if len(args) > 2 {
			return usageError(stderr, nil, "help accepts at most one command")
		}
		command := ""
		if len(args) == 2 {
			command = args[1]
		}
		if !writeHelp(stdout, command) {
			return usageError(stderr, nil, fmt.Sprintf("unknown command %q", command))
		}
		return 0
	}
	command, rest := args[0], args[1:]
	if len(rest) == 1 && (rest[0] == "--help" || rest[0] == "-h") {
		if !writeHelp(stdout, command) {
			return usageError(stderr, nil, fmt.Sprintf("unknown command %q", command))
		}
		return 0
	}
	switch command {
	case "render":
		return runRender(ctx, rest, stdin, stdout, stderr)
	case "preview":
		return runPreview(rest, stdout, stderr)
	case "task":
		return runTask(rest, stdout, stderr)
	case "config":
		return runConfig(rest, stdin, stdout, stderr)
	case "install":
		return runInstall(rest, stdout, stderr)
	case "explain":
		return runExplain(rest, stdout, stderr)
	case "doctor":
		return runDoctor(rest, stdout, stderr)
	case "version", "--version", "-v":
		fmt.Fprintln(stdout, Version)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", command)
		return 2
	}
}

func runRender(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	pos, opts, flags, err := parse(args)
	if err != nil || len(pos) != 0 {
		return usageError(stderr, err, "render accepts options only")
	}
	if err := validateCommand(opts, flags, []string{"runtime", "width", "format", "project", "home"}, []string{"ascii", "no-color"}); err != nil {
		return usageError(stderr, err, "")
	}
	project, home, err := roots(opts)
	if err != nil {
		return usageError(stderr, err, "")
	}
	runtimeName := value(opts, "runtime", "claude")
	if runtimeName == "auto" {
		runtimeName = "claude"
	}
	if runtimeName != "claude" && runtimeName != "generic" {
		return usageError(stderr, nil, "runtime must be auto, claude, or generic")
	}
	now := time.Now().UTC()
	var snapshot status.Snapshot
	if runtimeName == "claude" {
		snapshot, err = adapter.ParseClaude(stdin, now)
	} else {
		data, readErr := io.ReadAll(io.LimitReader(stdin, adapter.MaxClaudeInputBytes+1))
		if readErr != nil {
			err = readErr
		} else if len(data) > adapter.MaxClaudeInputBytes {
			err = fmt.Errorf("generic status input exceeds %d bytes", adapter.MaxClaudeInputBytes)
		} else {
			decoder := json.NewDecoder(strings.NewReader(string(data)))
			if err = decoder.Decode(&snapshot); err == nil {
				var extra any
				if nextErr := decoder.Decode(&extra); nextErr != io.EOF {
					if nextErr == nil {
						err = fmt.Errorf("generic status input contains multiple JSON values")
					} else {
						err = nextErr
					}
				}
			}
		}
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	enrich(&snapshot, ctx, project, now)
	cfg, err := config.Load(project, home, config.RuntimeClaude, config.Overrides{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	width, err := integer(opts, "width", 120)
	if err != nil || width < 1 {
		return usageError(stderr, err, "width must be positive")
	}
	format := value(opts, "format", "plain")
	renderFormat := format
	if format == "json" {
		renderFormat = "plain"
	}
	result, err := render.Render(snapshot, cfg, render.Options{Width: width, Format: renderFormat, Color: renderFormat == "ansi" && !flags["no-color"] && os.Getenv("NO_COLOR") == "", ASCII: flags["ascii"], HomeDir: home, UserName: filepath.Base(home), Now: now})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if format == "json" {
		return writeJSON(stdout, stderr, result)
	}
	fmt.Fprintln(stdout, result.Output)
	return 0
}

func runPreview(args []string, stdout, stderr io.Writer) int {
	pos, opts, flags, err := parse(args)
	if err != nil || len(pos) != 0 {
		return usageError(stderr, err, "preview accepts options only")
	}
	if err := validateCommand(opts, flags, []string{"preset", "width"}, []string{"ascii"}); err != nil {
		return usageError(stderr, err, "")
	}
	preset := value(opts, "preset", "standard")
	widths := map[string]int{"wide": 160, "standard": 120, "compact": 80, "minimal": 40}
	width, ok := widths[preset]
	if !ok {
		fmt.Fprintf(stderr, "unknown preset %q\n", preset)
		return 2
	}
	if raw := opts["width"]; raw != "" {
		width, err = strconv.Atoi(raw)
		if err != nil || width < 1 {
			return usageError(stderr, err, "width must be positive")
		}
	}
	cfg := defaultPreviewConfig()
	result, err := render.Render(previewSnapshot(), cfg, render.Options{Width: width, Format: "plain", ASCII: flags["ascii"]})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintln(stdout, result.Output)
	return 0
}

func runTask(args []string, stdout, stderr io.Writer) int {
	pos, opts, flags, err := parse(args)
	if err != nil || len(pos) == 0 {
		return usageError(stderr, err, "task requires set|step|block|done|clear|show")
	}
	project := value(opts, "project", "")
	if project == "" {
		project, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err)
			return 2
		}
	}
	now := time.Now().UTC()
	var task status.Task
	switch pos[0] {
	case "show":
		if len(pos) != 1 {
			return usageError(stderr, nil, "task show accepts no arguments")
		}
		if err := validateCommand(opts, flags, []string{"project"}, nil); err != nil {
			return usageError(stderr, err, "")
		}
		task, err = taskstate.Load(project)
	case "clear":
		if len(pos) != 1 {
			return usageError(stderr, nil, "task clear accepts no arguments")
		}
		if err := validateCommand(opts, flags, []string{"project"}, nil); err != nil {
			return usageError(stderr, err, "")
		}
		task, err = taskstate.Update(project, taskstate.Mutation{Kind: taskstate.Clear}, now)
	case "set":
		if len(pos) < 2 {
			return usageError(stderr, nil, "task set requires a title")
		}
		if err := validateCommand(opts, flags, []string{"project", "total", "phase"}, nil); err != nil {
			return usageError(stderr, err, "")
		}
		total, parseErr := integer(opts, "total", 0)
		if parseErr != nil {
			return usageError(stderr, parseErr, "")
		}
		task, err = taskstate.Update(project, taskstate.Mutation{Kind: taskstate.Set, Title: strings.Join(pos[1:], " "), Phase: opts["phase"], TotalSteps: total, SourceRuntime: "signalrail"}, now)
	case "step":
		if len(pos) != 1 {
			return usageError(stderr, nil, "task step accepts no arguments")
		}
		if err := validateCommand(opts, flags, []string{"project", "step"}, nil); err != nil {
			return usageError(stderr, err, "")
		}
		delta, parseErr := integer(opts, "step", 1)
		if parseErr != nil {
			return usageError(stderr, parseErr, "")
		}
		task, err = taskstate.Update(project, taskstate.Mutation{Kind: taskstate.Step, Step: delta}, now)
	case "block":
		if err := validateCommand(opts, flags, []string{"project", "note"}, nil); err != nil {
			return usageError(stderr, err, "")
		}
		note := value(opts, "note", strings.Join(pos[1:], " "))
		task, err = taskstate.Update(project, taskstate.Mutation{Kind: taskstate.Block, BlockerNote: note}, now)
	case "done":
		if len(pos) != 1 {
			return usageError(stderr, nil, "task done accepts no arguments")
		}
		if err := validateCommand(opts, flags, []string{"project"}, nil); err != nil {
			return usageError(stderr, err, "")
		}
		task, err = taskstate.Update(project, taskstate.Mutation{Kind: taskstate.Done}, now)
	default:
		return usageError(stderr, nil, "unknown task action")
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	if pos[0] == "clear" {
		fmt.Fprintln(stdout, "Task state cleared")
		return 0
	}
	return writeJSON(stdout, stderr, task)
}

func runConfig(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	pos, opts, flags, err := parse(args)
	if err != nil || len(pos) != 0 {
		return usageError(stderr, err, "config accepts options only")
	}
	if err := validateCommand(opts, flags, []string{"lang", "scope", "project", "home"}, nil); err != nil {
		return usageError(stderr, err, "")
	}
	lang := value(opts, "lang", "en")
	if lang != "en" && lang != "zh-CN" {
		fmt.Fprintln(stderr, "language must be en or zh-CN")
		return 2
	}
	project, home, rootErr := roots(opts)
	if rootErr != nil {
		return usageError(stderr, rootErr, "")
	}
	scope := value(opts, "scope", "project")
	if scope != "project" && scope != "user" {
		return usageError(stderr, nil, "scope must be user or project")
	}
	path := filepath.Join(project, ".signalrail.toml")
	if scope == "user" {
		path = config.UserPath(home)
	}
	if lang == "zh-CN" {
		fmt.Fprint(stdout, "SignalRail 设置\n1) 紧凑\n2) 标准\n3) 详细\nq) 取消\n请选择 [2]: ")
	} else {
		fmt.Fprint(stdout, "SignalRail Settings\n1) Compact\n2) Standard\n3) Detailed\nq) Cancel\nSelect [2]: ")
	}
	reader := bufio.NewReader(stdin)
	line, readErr := reader.ReadString('\n')
	if readErr != nil && readErr != io.EOF {
		fmt.Fprintln(stderr, readErr)
		return 2
	}
	choice := strings.TrimSpace(line)
	if choice == "q" || choice == "Q" {
		return 0
	}
	if choice == "" {
		choice = "2"
	}
	sets := map[string][]string{"1": {"model", "task", "context"}, "2": {"model", "project", "task", "progress", "context"}, "3": {"model", "project", "task", "progress", "context", "cost"}}
	segments, ok := sets[choice]
	if !ok {
		fmt.Fprintln(stderr, "selection must be 1, 2, 3, or q")
		return 2
	}
	content := "version = 1\nsegments = [" + quoteList(segments) + "]\n\n[cost]\nenabled = " + strconv.FormatBool(choice == "3") + "\ncurrency = \"USD\"\n"
	if lang == "zh-CN" {
		fmt.Fprintf(stdout, "\n预览:\n%s保存? [y/N]: ", content)
	} else {
		fmt.Fprintf(stdout, "\nPreview:\n%sSave? [y/N]: ", content)
	}
	confirm, confirmErr := reader.ReadString('\n')
	if confirmErr != nil && confirmErr != io.EOF {
		fmt.Fprintln(stderr, confirmErr)
		return 2
	}
	answer := strings.ToLower(strings.TrimSpace(confirm))
	if answer != "y" && answer != "yes" && answer != "是" {
		return 0
	}
	if err := atomicWrite(path, []byte(content), 0o600); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintf(stdout, "\nSaved %s\n", path)
	return 0
}

func runInstall(args []string, stdout, stderr io.Writer) int {
	pos, opts, flags, err := parse(args)
	if err != nil || len(pos) != 1 {
		return usageError(stderr, err, "install requires claude or codex")
	}
	if err := validateCommand(opts, flags, []string{"scope", "home", "project", "items"}, []string{"dry-run", "strict"}); err != nil {
		return usageError(stderr, err, "")
	}
	runtimeName := pos[0]
	if runtimeName != "claude" && runtimeName != "codex" {
		return usageError(stderr, nil, "install requires claude or codex")
	}
	project, home, err := roots(opts)
	if err != nil {
		return usageError(stderr, err, "")
	}
	scope := value(opts, "scope", "user")
	if scope != "user" && scope != "project" {
		return usageError(stderr, nil, "scope must be user or project")
	}
	base := home
	if scope == "project" {
		base = project
	}
	var change install.Change
	var warnings []install.Warning
	if runtimeName == "claude" {
		change, err = install.PlanClaude(filepath.Join(base, ".claude", "settings.json"), "signalrail render --runtime claude --format ansi")
	} else {
		items := strings.Split(value(opts, "items", "model,project,branch,progress,context"), ",")
		change, warnings, err = install.PlanCodex(filepath.Join(base, ".codex", "config.toml"), items, []string{"activity", "project"})
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	for _, warning := range warnings {
		fmt.Fprintf(stderr, "%s: %s\n", warning.Code, warning.Message)
	}
	if flags["strict"] && len(warnings) > 0 {
		return 3
	}
	if flags["dry-run"] {
		fmt.Fprint(stdout, string(change.After))
		return 0
	}
	if err := change.Apply(true); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	fmt.Fprintf(stdout, "Installed %s configuration at %s\n", runtimeName, change.Path)
	return 0
}

func runExplain(args []string, stdout, stderr io.Writer) int {
	pos, opts, flags, err := parse(args)
	if err != nil || len(pos) != 0 {
		return usageError(stderr, err, "explain accepts options only")
	}
	if err := validateCommand(opts, flags, []string{"project", "home"}, []string{"json"}); err != nil {
		return usageError(stderr, err, "")
	}
	project, home, err := roots(opts)
	if err != nil {
		return usageError(stderr, err, "")
	}
	now := time.Now().UTC()
	snapshot := localSnapshot(project, now)
	cfg, err := config.Load(project, home, config.RuntimeClaude, config.Overrides{})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	result, err := render.Render(snapshot, cfg, render.Options{Width: 120, Format: "plain", HomeDir: home, UserName: filepath.Base(home)})
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	explanation := diagnostics.Explain(snapshot, result, cfg.Segments, now)
	if flags["json"] {
		return writeJSON(stdout, stderr, explanation)
	}
	fmt.Fprintln(stdout, explanation.Output)
	for _, item := range explanation.Segments {
		fmt.Fprintf(stdout, "%s included=%t confidence=%s freshness=%s source=%s age_ms=%d\n", item.Name, item.Included, item.Confidence, item.Freshness, item.Source.Provider, item.AgeMS)
	}
	return 0
}

func runDoctor(args []string, stdout, stderr io.Writer) int {
	pos, opts, flags, err := parse(args)
	if err != nil || len(pos) != 0 {
		return usageError(stderr, err, "doctor accepts options only")
	}
	if err := validateCommand(opts, flags, []string{"project", "home"}, []string{"json"}); err != nil {
		return usageError(stderr, err, "")
	}
	project, home, err := roots(opts)
	if err != nil {
		return usageError(stderr, err, "")
	}
	report, code := diagnostics.Doctor(home, project)
	if flags["json"] {
		if writeJSON(stdout, stderr, report) != 0 {
			return 2
		}
		return code
	}
	for _, check := range report.Checks {
		fmt.Fprintf(stdout, "%s [%s] %s\n", check.Name, check.Status, check.Message)
	}
	return code
}

func enrich(snapshot *status.Snapshot, ctx context.Context, project string, now time.Time) {
	git := provider.Git(ctx, project)
	if snapshot.Project.Value.Name == "" {
		snapshot.Project.Value.Name = git.Name
	}
	snapshot.Project.Value.Branch = git.Branch
	snapshot.Project.Value.Dirty = git.Dirty
	if task, err := taskstate.Load(project); err == nil && task.Title != "" {
		snapshot.Task = status.Datum[status.Task]{Value: task, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: status.Source{Provider: "signalrail", Detail: "project-task-state"}, ObservedAt: task.UpdatedAt}
	}
	_ = now
}

func localSnapshot(project string, now time.Time) status.Snapshot {
	git := provider.Git(context.Background(), project)
	snapshot := status.Snapshot{
		Model: unavailableModel(now), Project: status.Datum[status.Project]{Value: git, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: status.Source{Provider: "git"}, ObservedAt: now},
		Task: unavailableTask(now), Context: unavailableContext(now), Cost: unavailableCost(now),
	}
	if task, err := taskstate.Load(project); err == nil && task.Title != "" {
		snapshot.Task = status.Datum[status.Task]{Value: task, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: status.Source{Provider: "signalrail"}, ObservedAt: task.UpdatedAt}
	}
	return snapshot
}

func previewSnapshot() status.Snapshot {
	now := time.Now().UTC()
	return status.Snapshot{
		Model:   status.Datum[status.Model]{Value: status.Model{Name: "GPT-5.5", Effort: "high"}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, ObservedAt: now},
		Project: status.Datum[status.Project]{Value: status.Project{Name: "signalrail", Branch: "main"}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, ObservedAt: now},
		Task:    status.Datum[status.Task]{Value: status.Task{Title: "Build renderer", Step: 3, TotalSteps: 7, State: "active"}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, ObservedAt: now},
		Context: status.Datum[status.Context]{Value: status.Context{UsedPercent: 62, LeftLabel: "38%"}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, ObservedAt: now},
		Cost:    status.Datum[status.Cost]{Value: status.Cost{Amount: 1.24, Currency: "USD"}, Confidence: status.ConfidenceEstimated, Freshness: status.FreshnessFresh, ObservedAt: now},
	}
}

func defaultPreviewConfig() config.Config {
	return config.Config{Segments: []status.SegmentName{status.SegmentModel, status.SegmentProject, status.SegmentTask, status.SegmentProgress, status.SegmentContext, status.SegmentCost}, Privacy: config.Privacy{RedactUser: true, RedactPaths: true}, Cost: config.Cost{Enabled: true, Currency: "USD"}}
}

func unavailableModel(now time.Time) status.Datum[status.Model] {
	return status.Datum[status.Model]{Confidence: status.ConfidenceUnavailable, Freshness: status.FreshnessFresh, ObservedAt: now}
}
func unavailableTask(now time.Time) status.Datum[status.Task] {
	return status.Datum[status.Task]{Confidence: status.ConfidenceUnavailable, Freshness: status.FreshnessFresh, ObservedAt: now}
}
func unavailableContext(now time.Time) status.Datum[status.Context] {
	return status.Datum[status.Context]{Confidence: status.ConfidenceUnavailable, Freshness: status.FreshnessFresh, ObservedAt: now}
}
func unavailableCost(now time.Time) status.Datum[status.Cost] {
	return status.Datum[status.Cost]{Confidence: status.ConfidenceUnavailable, Freshness: status.FreshnessFresh, ObservedAt: now}
}

func parse(args []string) ([]string, map[string]string, map[string]bool, error) {
	values := map[string]bool{"runtime": true, "width": true, "format": true, "project": true, "home": true, "preset": true, "lang": true, "total": true, "phase": true, "step": true, "note": true, "scope": true, "items": true}
	booleans := map[string]bool{"json": true, "dry-run": true, "strict": true, "ascii": true, "no-color": true}
	var pos []string
	opts := map[string]string{}
	flags := map[string]bool{}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			pos = append(pos, arg)
			continue
		}
		name := strings.TrimPrefix(arg, "--")
		if booleans[name] {
			flags[name] = true
			continue
		}
		if !values[name] {
			return nil, nil, nil, fmt.Errorf("unknown option --%s", name)
		}
		if i+1 >= len(args) {
			return nil, nil, nil, fmt.Errorf("option --%s requires a value", name)
		}
		if strings.HasPrefix(args[i+1], "--") {
			return nil, nil, nil, fmt.Errorf("option --%s requires a value", name)
		}
		i++
		opts[name] = args[i]
	}
	return pos, opts, flags, nil
}

func validateCommand(opts map[string]string, flags map[string]bool, allowedOpts, allowedFlags []string) error {
	allowedValue := map[string]bool{}
	for _, name := range allowedOpts {
		allowedValue[name] = true
	}
	for name := range opts {
		if !allowedValue[name] {
			return fmt.Errorf("option --%s is not valid for this command", name)
		}
	}
	allowedBool := map[string]bool{}
	for _, name := range allowedFlags {
		allowedBool[name] = true
	}
	for name := range flags {
		if !allowedBool[name] {
			return fmt.Errorf("option --%s is not valid for this command", name)
		}
	}
	return nil
}

func roots(opts map[string]string) (string, string, error) {
	project := opts["project"]
	home := opts["home"]
	var err error
	if project == "" {
		project, err = os.Getwd()
		if err != nil {
			return "", "", err
		}
	}
	if home == "" {
		home, err = os.UserHomeDir()
		if err != nil {
			return "", "", err
		}
	}
	return project, home, nil
}
func value(opts map[string]string, name, fallback string) string {
	if opts[name] != "" {
		return opts[name]
	}
	return fallback
}
func integer(opts map[string]string, name string, fallback int) (int, error) {
	if opts[name] == "" {
		return fallback, nil
	}
	return strconv.Atoi(opts[name])
}
func usageError(stderr io.Writer, err error, message string) int {
	if err != nil {
		fmt.Fprintln(stderr, err)
	} else if message != "" {
		fmt.Fprintln(stderr, message)
	}
	return 2
}
func writeHelp(stdout io.Writer, command string) bool {
	usage := map[string]string{
		"render":  "signalrail render [--runtime auto|claude|generic] [--width N] [--format ansi|plain|json]",
		"preview": "signalrail preview [--preset wide|standard|compact|minimal] [--width N]",
		"config":  "signalrail config [--lang en|zh-CN] [--scope user|project]",
		"task":    "signalrail task set|step|block|done|clear|show [options]",
		"install": "signalrail install claude|codex [--scope user|project] [--dry-run]",
		"explain": "signalrail explain [--json]",
		"doctor":  "signalrail doctor [--json]",
		"version": "signalrail version",
	}
	if command != "" {
		line, ok := usage[command]
		if !ok {
			return false
		}
		fmt.Fprintf(stdout, "Usage:\n  %s\n", line)
		return true
	}
	fmt.Fprintln(stdout, "Usage:\n  signalrail <command> [options]\n\nCommands:")
	summaries := map[string]string{
		"render": "render runtime status input", "preview": "preview adaptive UI presets",
		"config": "configure user or project policy", "task": "manage shared project task state",
		"install": "install Claude or Codex integration", "explain": "explain segment provenance",
		"doctor": "validate configuration and integrations", "version": "print the version",
	}
	for _, name := range []string{"render", "preview", "config", "task", "install", "explain", "doctor", "version"} {
		fmt.Fprintf(stdout, "  %-8s %s\n", name, summaries[name])
	}
	return true
}
func writeJSON(stdout, stderr io.Writer, value any) int {
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(value); err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}
	return 0
}
func quoteList(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = strconv.Quote(value)
	}
	return strings.Join(quoted, ", ")
}
func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return fmt.Errorf("config target must be a regular file")
		}
		mode = info.Mode().Perm()
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".signalrail-config-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if err := tmp.Chmod(mode); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(name, path); err != nil {
		return err
	}
	if runtime.GOOS != "windows" {
		dir, err := os.Open(filepath.Dir(path))
		if err != nil {
			return err
		}
		syncErr := dir.Sync()
		closeErr := dir.Close()
		if syncErr != nil {
			return syncErr
		}
		if closeErr != nil {
			return closeErr
		}
	}
	return nil
}
