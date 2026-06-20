package config

import "github.com/LAwLi3t-CN/signalrail/internal/status"

const CurrentVersion = 1

type Runtime string

const (
	RuntimeClaude Runtime = "claude"
	RuntimeCodex  Runtime = "codex"
)

type Config struct {
	Version  int
	Segments []status.SegmentName
	Privacy  Privacy
	Task     Task
	Context  Context
	Cost     Cost
}

type Privacy struct {
	RedactUser        bool     `toml:"redact_user"`
	RedactPaths       bool     `toml:"redact_paths"`
	SensitiveBranches []string `toml:"sensitive_branches"`
}

type Task struct {
	DefaultLabel string `toml:"default_label"`
	ShowPhase    bool   `toml:"show_phase"`
}

type Context struct {
	Label string `toml:"label"`
}

type Cost struct {
	Enabled  bool   `toml:"enabled"`
	Currency string `toml:"currency"`
}

type Overrides struct {
	Segments []status.SegmentName
	Privacy  PrivacyOverrides
	Task     TaskOverrides
	Context  ContextOverrides
	Cost     CostOverrides
}

type PrivacyOverrides struct {
	RedactUser        *bool
	RedactPaths       *bool
	SensitiveBranches *[]string
}

type TaskOverrides struct {
	DefaultLabel *string
	ShowPhase    *bool
}

type ContextOverrides struct {
	Label *string
}

type CostOverrides struct {
	Enabled  *bool
	Currency *string
}

type fileConfig struct {
	Version  *int                  `toml:"version"`
	Segments *[]status.SegmentName `toml:"segments"`
	Privacy  *privacyConfig        `toml:"privacy"`
	Task     *taskConfig           `toml:"task"`
	Context  *contextConfig        `toml:"context"`
	Cost     *costConfig           `toml:"cost"`
	Runtime  *runtimeConfig        `toml:"runtime"`
}

type runtimeConfig struct {
	Claude *runtimeLayer `toml:"claude"`
	Codex  *runtimeLayer `toml:"codex"`
}

type runtimeLayer struct {
	Segments *[]status.SegmentName `toml:"segments"`
	Privacy  *privacyConfig        `toml:"privacy"`
	Task     *taskConfig           `toml:"task"`
	Context  *contextConfig        `toml:"context"`
	Cost     *costConfig           `toml:"cost"`
}

type privacyConfig struct {
	RedactUser        *bool     `toml:"redact_user"`
	RedactPaths       *bool     `toml:"redact_paths"`
	SensitiveBranches *[]string `toml:"sensitive_branches"`
}

type taskConfig struct {
	DefaultLabel *string `toml:"default_label"`
	ShowPhase    *bool   `toml:"show_phase"`
}

type contextConfig struct {
	Label *string `toml:"label"`
}

type costConfig struct {
	Enabled  *bool   `toml:"enabled"`
	Currency *string `toml:"currency"`
}
