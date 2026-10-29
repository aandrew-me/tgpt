package providers

import "testing"

func TestSupportsTools(t *testing.T) {
	supported := []string{"openai", "opencode", "gemini", "groq", "deepseek", "ollama", "litellm", "omniroute", "openrouter", "anyapi", "atlascloud", "atmorouter", "apiroute", "cheaperinference", "demonroute", "pollinations", "runcrate", ""}
	for _, p := range supported {
		if !SupportsTools(p) {
			t.Errorf("expected provider %q to support tools", p)
		}
	}

	unsupported := []string{"aihorde", "deepseek-web", "fx", "isou", "koboldai", "minimax", "ollamacloud", "powerbrain", "invalid"}
	for _, p := range unsupported {
		if SupportsTools(p) {
			t.Errorf("expected provider %q NOT to support tools", p)
		}
	}

	if !IsValidProvider("deepseek-web") {
		t.Errorf("expected deepseek-web to be a valid provider")
	}

	if !IsValidProvider("fx") {
		t.Errorf("expected fx to be a valid provider")
	}
}

func TestGetMainTextKeepsContentContainingDataPrefix(t *testing.T) {
	line := `data: {"choices":[{"delta":{"content":"the header is data: x"}}]}`
	for _, p := range []string{"deepseek", "gemini", "groq", "minimax", "openai", "opencode", "pollinations"} {
		if got := GetMainText(line, p, ""); got != "the header is data: x" {
			t.Errorf("provider %q: got %q", p, got)
		}
	}
}

func TestKoboldAIKeepsTokenContainingDataPrefix(t *testing.T) {
	line := `data: {"token":"the header is data: x"}`
	if got := GetMainText(line, "koboldai", ""); got != "the header is data: x" {
		t.Errorf("got %q", got)
	}
}
