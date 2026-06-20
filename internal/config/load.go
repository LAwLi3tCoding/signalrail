package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/LAwLi3t-CN/signalrail/internal/status"
	toml "github.com/pelletier/go-toml/v2"
)

func Load(projectDir, userDir string, runtime Runtime, overrides Overrides) (Config, error) {
	cfg := defaultConfig()
	applyRuntimeLayer(&cfg, defaultRuntimeLayer(runtime))

	userConfig, err := loadFile(UserPath(userDir), "user")
	if err != nil {
		return Config{}, err
	}
	projectConfig, err := loadFile(projectPath(projectDir), "project")
	if err != nil {
		return Config{}, err
	}

	applyFileConfig(&cfg, userConfig)
	applyFileConfig(&cfg, projectConfig)

	applyRuntimeConfig(&cfg, userConfig, runtime)
	applyRuntimeConfig(&cfg, projectConfig, runtime)
	applyOverrides(&cfg, overrides)

	return cfg, nil
}

func defaultConfig() Config {
	return Config{
		Version: CurrentVersion,
		Segments: []status.SegmentName{
			status.SegmentModel,
			status.SegmentProject,
			status.SegmentTask,
			status.SegmentProgress,
			status.SegmentContext,
		},
		Privacy: Privacy{
			RedactUser:  true,
			RedactPaths: true,
		},
		Task: Task{
			DefaultLabel: "Task",
			ShowPhase:    true,
		},
		Context: Context{
			Label: "left",
		},
		Cost: Cost{
			Enabled:  false,
			Currency: "USD",
		},
	}
}

func defaultRuntimeLayer(runtime Runtime) *runtimeLayer {
	switch runtime {
	case RuntimeClaude:
		return nil
	case RuntimeCodex:
		segments := []status.SegmentName{
			status.SegmentModel,
			status.SegmentProject,
			status.SegmentProgress,
			status.SegmentContext,
		}
		return &runtimeLayer{Segments: &segments}
	default:
		return nil
	}
}

func UserPath(userDir string) string {
	if userDir == "" {
		return ""
	}
	if xdg := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "signalrail", "config.toml")
	}
	return filepath.Join(userDir, ".config", "signalrail", "config.toml")
}

func projectPath(projectDir string) string {
	if projectDir == "" {
		return ""
	}
	return filepath.Join(projectDir, ".signalrail.toml")
}

func loadFile(path, label string) (*fileConfig, error) {
	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s config: %w", label, err)
	}

	var cfg fileConfig
	decoder := toml.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode %s config %s: %w", label, path, err)
	}

	return &cfg, nil
}

func applyFileConfig(cfg *Config, fc *fileConfig) {
	if fc == nil {
		return
	}

	if fc.Version != nil {
		cfg.Version = *fc.Version
	}
	if fc.Segments != nil {
		cfg.Segments = append([]status.SegmentName(nil), (*fc.Segments)...)
	}
	if fc.Privacy != nil {
		applyPrivacy(&cfg.Privacy, fc.Privacy)
	}
	if fc.Task != nil {
		applyTask(&cfg.Task, fc.Task)
	}
	if fc.Context != nil {
		applyContext(&cfg.Context, fc.Context)
	}
	if fc.Cost != nil {
		applyCost(&cfg.Cost, fc.Cost)
	}
}

func applyRuntimeConfig(cfg *Config, fc *fileConfig, runtime Runtime) {
	if fc == nil || fc.Runtime == nil {
		return
	}
	applyRuntimeLayer(cfg, selectRuntimeLayer(fc.Runtime, runtime))
}

func applyRuntimeLayer(cfg *Config, layer *runtimeLayer) {
	if layer == nil {
		return
	}
	if layer.Segments != nil {
		cfg.Segments = append([]status.SegmentName(nil), (*layer.Segments)...)
	}
	if layer.Privacy != nil {
		applyPrivacy(&cfg.Privacy, layer.Privacy)
	}
	if layer.Task != nil {
		applyTask(&cfg.Task, layer.Task)
	}
	if layer.Context != nil {
		applyContext(&cfg.Context, layer.Context)
	}
	if layer.Cost != nil {
		applyCost(&cfg.Cost, layer.Cost)
	}
}

func selectRuntimeLayer(runtimeConfig *runtimeConfig, runtime Runtime) *runtimeLayer {
	switch runtime {
	case RuntimeClaude:
		return runtimeConfig.Claude
	case RuntimeCodex:
		return runtimeConfig.Codex
	default:
		return nil
	}
}

func applyOverrides(cfg *Config, overrides Overrides) {
	if overrides.Segments != nil {
		cfg.Segments = append([]status.SegmentName(nil), overrides.Segments...)
	}
	if overrides.Privacy.RedactUser != nil {
		cfg.Privacy.RedactUser = *overrides.Privacy.RedactUser
	}
	if overrides.Privacy.RedactPaths != nil {
		cfg.Privacy.RedactPaths = *overrides.Privacy.RedactPaths
	}
	if overrides.Privacy.SensitiveBranches != nil {
		cfg.Privacy.SensitiveBranches = append([]string(nil), (*overrides.Privacy.SensitiveBranches)...)
	}
	if overrides.Task.DefaultLabel != nil {
		cfg.Task.DefaultLabel = *overrides.Task.DefaultLabel
	}
	if overrides.Task.ShowPhase != nil {
		cfg.Task.ShowPhase = *overrides.Task.ShowPhase
	}
	if overrides.Context.Label != nil {
		cfg.Context.Label = *overrides.Context.Label
	}
	if overrides.Cost.Enabled != nil {
		cfg.Cost.Enabled = *overrides.Cost.Enabled
	}
	if overrides.Cost.Currency != nil {
		cfg.Cost.Currency = *overrides.Cost.Currency
	}
}

func applyPrivacy(dst *Privacy, src *privacyConfig) {
	if src.RedactUser != nil {
		dst.RedactUser = *src.RedactUser
	}
	if src.RedactPaths != nil {
		dst.RedactPaths = *src.RedactPaths
	}
	if src.SensitiveBranches != nil {
		dst.SensitiveBranches = append([]string(nil), (*src.SensitiveBranches)...)
	}
}

func applyTask(dst *Task, src *taskConfig) {
	if src.DefaultLabel != nil {
		dst.DefaultLabel = *src.DefaultLabel
	}
	if src.ShowPhase != nil {
		dst.ShowPhase = *src.ShowPhase
	}
}

func applyContext(dst *Context, src *contextConfig) {
	if src.Label != nil {
		dst.Label = *src.Label
	}
}

func applyCost(dst *Cost, src *costConfig) {
	if src.Enabled != nil {
		dst.Enabled = *src.Enabled
	}
	if src.Currency != nil {
		dst.Currency = *src.Currency
	}
}
