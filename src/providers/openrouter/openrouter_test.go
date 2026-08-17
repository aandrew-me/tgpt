package openrouter

import (
	"encoding/json"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "router-key")
	t.Setenv("OPENROUTER_MODEL", "router-model")
	t.Setenv("OPENROUTER_BASE_URL", "https://example.com/v1/")

	params := structs.Params{}
	assert.Equal(t, "router-key", apiKey(params))
	assert.Equal(t, "router-model", model(params))
	assert.Equal(t, "https://example.com/v1/chat/completions", endpoint(params))

	params = structs.Params{ApiKey: "flag-key", ApiModel: "flag-model", Url: "https://override.test/chat"}
	assert.Equal(t, "flag-key", apiKey(params))
	assert.Equal(t, "flag-model", model(params))
	assert.Equal(t, "https://override.test/chat", endpoint(params))
}

func TestDefaults(t *testing.T) {
	t.Setenv("OPENROUTER_API_KEY", "")
	t.Setenv("AI_API_KEY", "")
	t.Setenv("OPENROUTER_MODEL", "")
	t.Setenv("OPENROUTER_URL", "")
	t.Setenv("OPENROUTER_BASE_URL", "")

	assert.Equal(t, defaultModel, model(structs.Params{}))
	assert.Equal(t, defaultURL, endpoint(structs.Params{}))
}

func TestRequestBody(t *testing.T) {
	body := RequestBody{
		Model:  defaultModel,
		Stream: true,
		Messages: []any{
			structs.DefaultMessage{Role: "user", Content: "Hello"},
		},
		Tools: []any{map[string]any{"type": "function"}},
	}

	encoded, err := json.Marshal(body)
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, defaultModel, decoded["model"])
	assert.Equal(t, true, decoded["stream"])
	assert.Len(t, decoded["messages"], 1)
	assert.Len(t, decoded["tools"], 1)
}

func TestGetMainText(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"content", `data: {"choices":[{"delta":{"content":"Hello"}}]}`, "Hello"},
		{"done", "data: [DONE]", ""},
		{"empty choices", `data: {"choices":[]}`, ""},
		{"malformed", "data: {", ""},
		{"not data", `{"choices":[{"delta":{"content":"ignored"}}]}`, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, GetMainText(tt.line))
		})
	}
}
