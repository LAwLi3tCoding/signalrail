package adapter

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"time"

	"github.com/LAwLi3t-CN/signalrail/internal/status"
)

const MaxClaudeInputBytes = 1 << 20

type claudePayload struct {
	Model struct {
		ID          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"model"`
	Workspace struct {
		CurrentDir string `json:"current_dir"`
		ProjectDir string `json:"project_dir"`
	} `json:"workspace"`
	Cost *struct {
		TotalCostUSD float64 `json:"total_cost_usd"`
	} `json:"cost"`
	ContextWindow *struct {
		ContextWindowSize   int     `json:"context_window_size"`
		UsedPercentage      float64 `json:"used_percentage"`
		RemainingPercentage float64 `json:"remaining_percentage"`
	} `json:"context_window"`
	Effort *struct {
		Level string `json:"level"`
	} `json:"effort"`
}

func ParseClaude(reader io.Reader, now time.Time) (status.Snapshot, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxClaudeInputBytes+1))
	if err != nil {
		return status.Snapshot{}, fmt.Errorf("read Claude status input: %w", err)
	}
	if len(data) > MaxClaudeInputBytes {
		return status.Snapshot{}, fmt.Errorf("Claude status input exceeds %d bytes", MaxClaudeInputBytes)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	var payload *claudePayload
	if err := decoder.Decode(&payload); err != nil {
		return status.Snapshot{}, fmt.Errorf("decode Claude status input: %w", err)
	}
	if err := ensureEOF(decoder); err != nil {
		return status.Snapshot{}, err
	}
	if payload == nil {
		return status.Snapshot{}, fmt.Errorf("Claude status input must be a JSON object")
	}

	observed := now.UTC()
	exact := func(detail string) status.Source { return status.Source{Provider: "claude", Detail: detail} }
	model := status.Model{Name: payload.Model.DisplayName}
	if payload.Effort != nil {
		model.Effort = payload.Effort.Level
	}
	snapshot := status.Snapshot{
		Model:   status.Datum[status.Model]{Value: model, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: exact("statusline-json"), ObservedAt: observed},
		Project: unavailable[status.Project](observed, "workspace"),
		Cost:    unavailable[status.Cost](observed, "cost"),
		Context: unavailable[status.Context](observed, "context-window"),
	}
	projectDir := payload.Workspace.ProjectDir
	if projectDir == "" {
		projectDir = payload.Workspace.CurrentDir
	}
	if projectDir != "" {
		snapshot.Project = status.Datum[status.Project]{Value: status.Project{Name: filepath.Base(filepath.Clean(projectDir))}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: exact("workspace"), ObservedAt: observed}
	}
	if payload.ContextWindow != nil {
		used := payload.ContextWindow.UsedPercentage
		remaining := payload.ContextWindow.RemainingPercentage
		snapshot.Context = status.Datum[status.Context]{Value: status.Context{UsedPercent: used, LeftLabel: formatPercent(remaining), WindowLabel: compactTokens(payload.ContextWindow.ContextWindowSize)}, Confidence: status.ConfidenceExact, Freshness: status.FreshnessFresh, Source: exact("context-window"), ObservedAt: observed}
	}
	if payload.Cost != nil {
		snapshot.Cost = status.Datum[status.Cost]{Value: status.Cost{Amount: payload.Cost.TotalCostUSD, Currency: "USD"}, Confidence: status.ConfidenceEstimated, Freshness: status.FreshnessFresh, Source: exact("client-cost-estimate"), ObservedAt: observed}
	}
	return snapshot, nil
}

func formatPercent(value float64) string { return fmt.Sprintf("%g%%", value) }

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("decode trailing Claude status input: %w", err)
	}
	return fmt.Errorf("Claude status input contains multiple JSON values")
}

func unavailable[T any](now time.Time, detail string) status.Datum[T] {
	return status.Datum[T]{Confidence: status.ConfidenceUnavailable, Freshness: status.FreshnessFresh, Source: status.Source{Provider: "claude", Detail: detail}, ObservedAt: now}
}

func compactTokens(tokens int) string {
	if tokens <= 0 {
		return ""
	}
	if tokens%1000 == 0 {
		return fmt.Sprintf("%dk", tokens/1000)
	}
	return fmt.Sprintf("%d", tokens)
}
