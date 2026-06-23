package install

import (
	"encoding/json"
	"fmt"
)

func PlanClaude(path, command string) (Change, error) {
	before, err := readOptional(path)
	if err != nil {
		return Change{}, fmt.Errorf("read Claude settings: %w", err)
	}
	return PlanClaudeBytes(path, before, command)
}

func PlanClaudeBytes(path string, before []byte, command string) (Change, error) {
	root := map[string]any{}
	if len(before) > 0 {
		if err := json.Unmarshal(before, &root); err != nil {
			return Change{}, fmt.Errorf("decode Claude settings: %w", err)
		}
		if root == nil {
			return Change{}, fmt.Errorf("Claude settings must be a JSON object")
		}
	}
	line := map[string]any{}
	if existing, ok := root["statusLine"].(map[string]any); ok {
		for key, value := range existing {
			line[key] = value
		}
	}
	line["type"] = "command"
	line["command"] = command
	line["padding"] = 0
	root["statusLine"] = line
	after, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return Change{}, fmt.Errorf("encode Claude settings: %w", err)
	}
	after = append(after, '\n')
	return Change{Path: path, Before: append([]byte(nil), before...), After: after}, nil
}
