package providers

import "testing"

func TestSupportsTools(t *testing.T) {
	supported := []string{"openai", "opencode", "gemini", "groq", "deepseek", "ollama", "litellm", "anyapi", "pollinations", ""}
	for _, p := range supported {
		if !SupportsTools(p) {
			t.Errorf("expected provider %q to support tools", p)
		}
	}

	unsupported := []string{"aihorde", "isou", "koboldai", "minimax", "ollamacloud", "powerbrain", "invalid"}
	for _, p := range unsupported {
		if SupportsTools(p) {
			t.Errorf("expected provider %q NOT to support tools", p)
		}
	}
}
