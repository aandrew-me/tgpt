package opencode

import (
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
)

func TestGetMainText(t *testing.T) {
	line := `data: {"choices":[{"delta":{"content":"Hello!"}}]}`
	got := GetMainText(line)
	if got != "Hello!" {
		t.Errorf("GetMainText() = %q, want %q", got, "Hello!")
	}
}

func TestDefaultToolsCount(t *testing.T) {
	if len(defaultTools) != 13 {
		t.Errorf("defaultTools count = %d, want 13", len(defaultTools))
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

func TestGenerateSessionID(t *testing.T) {
	id := generateSessionID()
	matched, err := regexp.MatchString(`^ses_[0-9a-f]{12}[0-9A-Za-z]{14}$`, id)
	if err != nil {
		t.Fatal(err)
	}
	if !matched {
		t.Errorf("generateSessionID() = %q, expected ses_<12 hex><14 base62>", id)
	}
}

func TestResolveSessionID(t *testing.T) {
	valid := "ses_e313ff36dffe2ANQjPqkE6EIuj"
	if got := resolveSessionID(valid); got != valid {
		t.Errorf("resolveSessionID(valid) = %q, want %q", got, valid)
	}

	first := resolveSessionID("interactive-thread-with-hyphens-12345")
	second := resolveSessionID("interactive-thread-with-hyphens-12345")
	if first != second {
		t.Errorf("resolveSessionID not stable for same ThreadID: %q != %q", first, second)
	}
	if !sessionIDPattern.MatchString(first) {
		t.Errorf("resolveSessionID(interactive) = %q, want valid ses_ format", first)
	}
}

func TestDefaultSystemPromptNeutralEnv(t *testing.T) {
	got := defaultSystemPrompt()
	if !strings.Contains(got, "# Harness") || !strings.Contains(got, "Working directory: .") {
		t.Errorf("defaultSystemPrompt() missing expected sections: %q", got)
	}
	if cwd, err := os.Getwd(); err == nil && cwd != "" && cwd != "." && strings.Contains(got, cwd) {
		t.Errorf("defaultSystemPrompt() leaked local cwd %q", cwd)
	}
}

func TestBuildRequestBodyToolChoice(t *testing.T) {
	noTools := buildRequestBody("big-pickle", "hi", structs.Params{})
	if noTools.ToolChoice != "none" {
		t.Errorf("ToolChoice with no params.Tools = %q, want 'none'", noTools.ToolChoice)
	}

	withTools := buildRequestBody("big-pickle", "hi", structs.Params{
		Tools: []any{map[string]any{"type": "function", "function": map[string]any{"name": "custom"}}},
	})
	if withTools.ToolChoice != "auto" {
		t.Errorf("ToolChoice with params.Tools = %q, want 'auto'", withTools.ToolChoice)
	}
}

func TestBunProfileSpec(t *testing.T) {
	id := bunProfile().GetClientHelloId()
	spec, err := id.ToSpec()
	if err != nil {
		t.Fatalf("ToSpec() failed: %v", err)
	}
	if len(spec.CipherSuites) != 17 {
		t.Errorf("CipherSuites len = %d, want 17", len(spec.CipherSuites))
	}
}

func TestURLFromEnv(t *testing.T) {
	want := "http://localhost:8080/v1/chat/completions"
	for _, in := range []string{"http://localhost:8080", "http://localhost:8080/"} {
		if got := urlFromEnv(in); got != want {
			t.Errorf("urlFromEnv(%q) = %q, want %q", in, got, want)
		}
	}
	if got := urlFromEnv(""); got != "" {
		t.Errorf("urlFromEnv(\"\") = %q, want empty", got)
	}
}
