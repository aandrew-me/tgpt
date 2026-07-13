package litellm

import (
	"encoding/json"
	"os"
	"strings"
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
			name:     "delta with empty content",
			input:    `data: {"id":"chatcmpl-123","choices":[{"delta":{"content":""}}]}`,
			expected: "",
		},
		{
			name:     "multi-word content",
			input:    `data: {"id":"chatcmpl-456","choices":[{"delta":{"content":"Hello, world!"}}]}`,
			expected: "Hello, world!",
		},
		{
			name:     "content with newline",
			input:    `data: {"id":"chatcmpl-789","choices":[{"delta":{"content":"\n"}}]}`,
			expected: "\n",
		},
		{
			name:     "content with unicode",
			input:    `data: {"id":"chatcmpl-abc","choices":[{"delta":{"content":"你好世界"}}]}`,
			expected: "你好世界",
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
	input := "What is 1+1?"
	params := structs.Params{
		Provider:     "litellm",
		ApiModel:     "openai/gpt-4o-mini",
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
		},
	}

	requestInfo.Messages = append(requestInfo.Messages, structs.DefaultMessage{
		Role:    "user",
		Content: input,
	})

	jsonBytes, err := json.Marshal(requestInfo)
	assert.Nil(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	assert.Nil(t, err)

	assert.Equal(t, "openai/gpt-4o-mini", decoded["model"])
	assert.Equal(t, true, decoded["stream"])

	messages := decoded["messages"].([]interface{})
	assert.Equal(t, 2, len(messages))

	sysMsg := messages[0].(map[string]interface{})
	assert.Equal(t, "system", sysMsg["role"])
	assert.Equal(t, "You are a helpful assistant", sysMsg["content"])

	userMsg := messages[1].(map[string]interface{})
	assert.Equal(t, "user", userMsg["role"])
	assert.Equal(t, "What is 1+1?", userMsg["content"])
}

func TestNewRequestWithPrevMessages(t *testing.T) {
	input := "Follow up question"
	prevMessages := []any{
		structs.DefaultMessage{
			Role:    "user",
			Content: "Previous question",
		},
		structs.DefaultMessage{
			Role:    "assistant",
			Content: "Previous answer",
		},
	}

	requestInfo := RequestBody{
		Model:  "anthropic/claude-haiku-4-5",
		Stream: true,
		Messages: []any{
			structs.DefaultMessage{
				Content: "system prompt",
				Role:    "system",
			},
		},
	}

	requestInfo.Messages = append(requestInfo.Messages, prevMessages...)
	requestInfo.Messages = append(requestInfo.Messages, structs.DefaultMessage{
		Role:    "user",
		Content: input,
	})

	jsonBytes, err := json.Marshal(requestInfo)
	assert.Nil(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	assert.Nil(t, err)

	messages := decoded["messages"].([]interface{})
	assert.Equal(t, 4, len(messages))

	assert.Equal(t, "system", messages[0].(map[string]interface{})["role"])
	assert.Equal(t, "user", messages[1].(map[string]interface{})["role"])
	assert.Equal(t, "Previous question", messages[1].(map[string]interface{})["content"])
	assert.Equal(t, "assistant", messages[2].(map[string]interface{})["role"])
	assert.Equal(t, "Previous answer", messages[2].(map[string]interface{})["content"])
	assert.Equal(t, "user", messages[3].(map[string]interface{})["role"])
	assert.Equal(t, "Follow up question", messages[3].(map[string]interface{})["content"])
}

func TestNewRequestCustomModel(t *testing.T) {
	requestInfo := RequestBody{
		Model:  "anthropic/claude-sonnet-4-6",
		Stream: true,
		Messages: []any{
			structs.DefaultMessage{
				Content: "test",
				Role:    "system",
			},
		},
	}

	jsonBytes, err := json.Marshal(requestInfo)
	assert.Nil(t, err)

	var decoded map[string]interface{}
	err = json.Unmarshal(jsonBytes, &decoded)
	assert.Nil(t, err)

	assert.Equal(t, "anthropic/claude-sonnet-4-6", decoded["model"])
}

func TestModelFromParams(t *testing.T) {
	params := structs.Params{ApiModel: "anthropic/claude-sonnet-4-6"}
	model := params.ApiModel
	if model == "" {
		model = os.Getenv("LITELLM_MODEL")
	}
	assert.Equal(t, "anthropic/claude-sonnet-4-6", model)
}

func TestModelFromEnv(t *testing.T) {
	t.Setenv("LITELLM_MODEL", "azure/gpt-4o")
	params := structs.Params{}
	model := params.ApiModel
	if model == "" {
		model = os.Getenv("LITELLM_MODEL")
	}
	assert.Equal(t, "azure/gpt-4o", model)
}

func TestModelRequiredWhenEmpty(t *testing.T) {
	params := structs.Params{}
	model := params.ApiModel
	if model == "" {
		model = os.Getenv("LITELLM_MODEL")
	}
	assert.Equal(t, "", model, "model should be empty when neither flag nor env var is set")
}

func TestApiKeyFromParams(t *testing.T) {
	apiKey := ""
	params := structs.Params{ApiKey: "test-key-123"}

	if params.ApiKey != "" {
		apiKey = params.ApiKey
	}
	assert.Equal(t, "test-key-123", apiKey)
}

func TestDefaultUrl(t *testing.T) {
	url := ""
	params := structs.Params{}

	if params.Url != "" {
		url = params.Url
	}
	if url == "" {
		url = os.Getenv("LITELLM_URL")
	}
	if url == "" {
		url = "http://localhost:4000/v1/chat/completions"
	}
	assert.Equal(t, "http://localhost:4000/v1/chat/completions", url)
}

func TestCustomUrl(t *testing.T) {
	url := ""
	params := structs.Params{Url: "https://my-proxy.example.com/v1/chat/completions"}

	if params.Url != "" {
		url = params.Url
	}
	assert.Equal(t, "https://my-proxy.example.com/v1/chat/completions", url)
}

func TestGetMainTextStreamParsing(t *testing.T) {
	streamData := `data: {"id":"chatcmpl-1","choices":[{"delta":{"content":"The"}}]}
data: {"id":"chatcmpl-1","choices":[{"delta":{"content":" answer"}}]}
data: {"id":"chatcmpl-1","choices":[{"delta":{"content":" is"}}]}
data: {"id":"chatcmpl-1","choices":[{"delta":{"content":" 2"}}]}
data: [DONE]`

	var fullText strings.Builder
	for _, line := range strings.Split(streamData, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		text := GetMainText(line)
		fullText.WriteString(text)
	}

	assert.Equal(t, "The answer is 2", fullText.String())
}
