package providers

import "testing"

func TestSupportsTools(t *testing.T) {
	supported := []string{"openai", "opencode", "gemini", "groq", "deepseek", "ollama", "litellm", "omniroute", "openrouter", "anyapi", "atlascloud", "pollinations", ""}
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
