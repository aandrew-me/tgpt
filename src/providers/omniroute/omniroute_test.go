package omniroute

import (
	"encoding/json"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/stretchr/testify/assert"
)

func TestGetMainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid streaming chunk",
			input:    `data: {"id":"chatcmpl-123","choices":[{"delta":{"content":"Hello"}}]}`,
			expected: "Hello",
		},
		{
			name:     "empty choices",
			input:    `data: {"id":"chatcmpl-123","choices":[]}`,
			expected: "",
		},
		{
			name:     "done signal",
			input:    `data: [DONE]`,
			expected: "",
		},
		{
			name:     "empty line",
			input:    "",
			expected: "",
		},
		{
			name:     "malformed json",
			input:    `data: {invalid json}`,
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetMainText(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewRequestBodyFormat(t *testing.T) {
	input := "Hello"
	params := structs.Params{
		Provider:     "omniroute",
		ApiModel:     "gpt-4o",
		SystemPrompt: "You are a helpful assistant",
	}

	requestInfo := RequestBody{
		Model:  params.ApiModel,
		Stream: true,
		Messages: []any{
			structs.DefaultMessage{
				Content: params.SystemPrompt,
				Role:    "system",
			},
			structs.DefaultMessage{
				Role:    "user",
				Content: input,
			},
		},
	}

	jsonBytes, err := json.Marshal(requestInfo)
	assert.Nil(t, err)

	var decoded map[string]any
	err = json.Unmarshal(jsonBytes, &decoded)
	assert.Nil(t, err)
	assert.Equal(t, "gpt-4o", decoded["model"])
	assert.Equal(t, true, decoded["stream"])
}
