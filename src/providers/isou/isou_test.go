package isou

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestBodyEscapesPrompt(t *testing.T) {
	prompt := "say \"hi\"\nnext\\line"
	raw, err := newRequestBody(prompt, "gpt-5.4-mini")
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got), "body must be valid JSON: %s", raw)
	assert.Equal(t, "gpt-5.4-mini", got["model"])
	assert.Equal(t, "openai", got["provider"])
	assert.Equal(t, "all", got["language"])
	assert.Equal(t, []any{"general"}, got["categories"])
	assert.Equal(t, "SEARXNG", got["engine"])

	msgs, ok := got["messages"].([]any)
	require.True(t, ok)
	require.Len(t, msgs, 1)
	msg := msgs[0].(map[string]any)
	assert.Equal(t, "user", msg["role"])
	assert.Equal(t, prompt, msg["content"])
}

func TestNewRequestBodyEscapesModel(t *testing.T) {
	raw, err := newRequestBody("hello", "model\"v1")
	require.NoError(t, err)

	var got map[string]any
	require.NoError(t, json.Unmarshal(raw, &got), "body must be valid JSON: %s", raw)
	assert.Equal(t, "model\"v1", got["model"])
	msgs := got["messages"].([]any)
	assert.Equal(t, "hello", msgs[0].(map[string]any)["content"])
}
