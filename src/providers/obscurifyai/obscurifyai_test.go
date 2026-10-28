package obscurifyai

import (
	"encoding/json"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration(t *testing.T) {
	t.Setenv("OBSCURIFY_API_KEY", "obscurify-key")
	t.Setenv("OBSCURIFY_MODEL", "obscurify-model")
	t.Setenv("OBSCURIFY_BASE_URL", "https://example.com/v1/")

	params := structs.Params{}
	assert.Equal(t, "obscurify-key", apiKey(params))
	assert.Equal(t, "obscurify-model", model(params))
	assert.Equal(t, "https://example.com/v1/chat/completions", endpoint(params))

	params = structs.Params{ApiKey: "flag-key", ApiModel: "flag-model", Url: "https://override.test/chat"}
	assert.Equal(t, "flag-key", apiKey(params))
	assert.Equal(t, "flag-model", model(params))
	assert.Equal(t, "https://override.test/chat", endpoint(params))
}

func TestDefaults(t *testing.T) {
	t.Setenv("OBSCURIFY_API_KEY", "")
	t.Setenv("AI_API_KEY", "")
	t.Setenv("OBSCURIFY_MODEL", "")
	t.Setenv("OBSCURIFY_URL", "")
	t.Setenv("OBSCURIFY_BASE_URL", "")

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
