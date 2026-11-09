package standardcompute

import (
	"encoding/json"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
)

func TestConfiguration(t *testing.T) {
	t.Setenv("STANDARDCOMPUTE_API_KEY", "standardcompute-key")
	t.Setenv("STANDARDCOMPUTE_MODEL", "standardcompute-model")
	t.Setenv("STANDARDCOMPUTE_BASE_URL", "https://example.com/v1/")

	params := structs.Params{}
	assert.Equal(t, "standardcompute-key", apiKey(params))
	assert.Equal(t, "standardcompute-model", model(params))
	assert.Equal(t, "https://example.com/v1/chat/completions", endpoint(params))

	params = structs.Params{ApiKey: "flag-key", ApiModel: "flag-model", Url: "https://override.test/chat"}
	assert.Equal(t, "flag-key", apiKey(params))
	assert.Equal(t, "flag-model", model(params))
	assert.Equal(t, "https://override.test/chat", endpoint(params))
}

func TestDefaults(t *testing.T) {
	t.Setenv("STANDARDCOMPUTE_API_KEY", "")
	t.Setenv("AI_API_KEY", "")
	t.Setenv("STANDARDCOMPUTE_MODEL", "")
	t.Setenv("STANDARDCOMPUTE_URL", "")
	t.Setenv("STANDARDCOMPUTE_BASE_URL", "")

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

func TestRequestBodyIncludesSamplingOptions(t *testing.T) {
	body := newRequestBody(nil, structs.Params{Temperature: "0.2", Top_p: "0.9"})
	encoded, err := json.Marshal(body)
	assert.NoError(t, err)

	var decoded map[string]any
	assert.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.Equal(t, 0.2, decoded["temperature"])
	assert.Equal(t, 0.9, decoded["top_p"])

	encoded, err = json.Marshal(newRequestBody(nil, structs.Params{}))
	assert.NoError(t, err)
	decoded = map[string]any{}
	assert.NoError(t, json.Unmarshal(encoded, &decoded))
	assert.NotContains(t, decoded, "temperature")
	assert.NotContains(t, decoded, "top_p")
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
