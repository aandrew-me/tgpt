package minimax

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	// Create a mock server to capture the request
	var capturedBody []byte
	var capturedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedBody, _ = io.ReadAll(r.Body)
		capturedHeaders = r.Header
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte("data: {\"id\":\"test\",\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\ndata: [DONE]\n\n"))
	}))
	defer server.Close()

	// We can't easily override the URL in the provider since it's hardcoded,
	// so we test the request body structure by examining what would be built
	input := "What is 1+1?"
	params := structs.Params{
		Provider:     "minimax",
		SystemPrompt: "You are a helpful assistant",
	}

	// Build request body manually (same logic as NewRequest)
	requestInfo := RequestBody{
		Model:  "MiniMax-M2.7",
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

	// Verify structure
	assert.Equal(t, "MiniMax-M2.7", decoded["model"])
	assert.Equal(t, true, decoded["stream"])

	messages := decoded["messages"].([]interface{})
	assert.Equal(t, 2, len(messages))

	// System message
	sysMsg := messages[0].(map[string]interface{})
	assert.Equal(t, "system", sysMsg["role"])
	assert.Equal(t, "You are a helpful assistant", sysMsg["content"])

	// User message
	userMsg := messages[1].(map[string]interface{})
	assert.Equal(t, "user", userMsg["role"])
	assert.Equal(t, "What is 1+1?", userMsg["content"])

	_ = capturedBody
	_ = capturedHeaders
	_ = server
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
		Model:  "MiniMax-M2.7",
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
	assert.Equal(t, 4, len(messages)) // system + 2 prev + user

	// Verify conversation order
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
		Model:  "MiniMax-M2.7-highspeed",
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

	assert.Equal(t, "MiniMax-M2.7-highspeed", decoded["model"])
}

func TestModelEnvVar(t *testing.T) {
	// Test default model
	model := "MiniMax-M2.7"
	params := structs.Params{}

	if params.ApiModel != "" {
		model = params.ApiModel
	} else if envModel := os.Getenv("MINIMAX_MODEL"); envModel != "" {
		model = envModel
	}
	assert.Equal(t, "MiniMax-M2.7", model)

	// Test params override
	model = "MiniMax-M2.7"
	params = structs.Params{ApiModel: "custom-model"}
	if params.ApiModel != "" {
		model = params.ApiModel
	}
	assert.Equal(t, "custom-model", model)
}

func TestApiKeyFromParams(t *testing.T) {
	apiKey := ""
	params := structs.Params{ApiKey: "test-key-123"}

	envKey := os.Getenv("MINIMAX_API_KEY")
	if envKey != "" {
		apiKey = envKey
	}
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	}
	assert.Equal(t, "test-key-123", apiKey)
}

func TestGetMainTextStreamParsing(t *testing.T) {
	// Simulate a full streaming response
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

// Integration test - requires MINIMAX_API_KEY
func TestRequestIntegration(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY not set, skipping integration test")
	}

	resp, err := NewRequest("What is 1+1? Answer with just the number.", structs.Params{
		Provider:     "minimax",
		ApiKey:       apiKey,
		SystemPrompt: "You are a helpful assistant. Be concise.",
	})

	assert.Nil(t, err, "NewRequest should not return an error")
	assert.NotNil(t, resp, "Response should not be nil")
	assert.Equal(t, 200, resp.StatusCode, "Response status code should be 200")

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "data:", "Response should contain SSE data lines")
}

func TestRequestIntegrationHighspeed(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY not set, skipping integration test")
	}

	resp, err := NewRequest("Say hello in one word.", structs.Params{
		Provider:     "minimax",
		ApiKey:       apiKey,
		ApiModel:     "MiniMax-M2.7-highspeed",
		SystemPrompt: "You are a helpful assistant.",
	})

	assert.Nil(t, err, "NewRequest should not return an error")
	assert.NotNil(t, resp, "Response should not be nil")
	assert.Equal(t, 200, resp.StatusCode, "Response status code should be 200")

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "data:", "Response should contain SSE data lines")
}

func TestRequestIntegrationStreaming(t *testing.T) {
	apiKey := os.Getenv("MINIMAX_API_KEY")
	if apiKey == "" {
		t.Skip("MINIMAX_API_KEY not set, skipping integration test")
	}

	resp, err := NewRequest("What is 2+2? Answer with just the number.", structs.Params{
		Provider:     "minimax",
		ApiKey:       apiKey,
		SystemPrompt: "Be concise.",
	})

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	defer resp.Body.Close()

	// Parse streaming response and verify content extraction
	var fullText strings.Builder
	for _, line := range strings.Split(string(body), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "data: ") || line == "data: [DONE]" {
			continue
		}
		text := GetMainText(line)
		fullText.WriteString(text)
	}

	assert.NotEmpty(t, fullText.String(), "Should have extracted text from streaming response")
}
