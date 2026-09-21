package opencode

import (
	"testing"
)

func TestGetMainText(t *testing.T) {
	line := `data: {"choices":[{"delta":{"content":"Hello!"}}]}`
	got := GetMainText(line)
	if got != "Hello!" {
		t.Errorf("GetMainText() = %q, want %q", got, "Hello!")
	}
}

func TestDefaultToolsCount(t *testing.T) {
	if len(defaultTools) != 12 {
		t.Errorf("defaultTools count = %d, want 12", len(defaultTools))
	}
}

func TestToolName(t *testing.T) {
	m := map[string]any{
		"type": "function",
		"function": map[string]any{
			"name": "glob",
		},
	}
	if name := toolName(m); name != "glob" {
		t.Errorf("toolName(m) = %q, want 'glob'", name)
	}

	type customFunction struct {
		Name string `json:"name"`
	}
	type customTool struct {
		Type     string         `json:"type"`
		Function customFunction `json:"function"`
	}
	st := customTool{Type: "function", Function: customFunction{Name: "grep"}}
	if name := toolName(st); name != "grep" {
		t.Errorf("toolName(st) = %q, want 'grep'", name)
	}
}

