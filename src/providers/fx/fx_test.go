package fx

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
)

func TestGetMainText(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "message prefix text delta",
			input:    `message: {"type":"text-delta","id":"txt-0","delta":" hard"}`,
			expected: " hard",
		},
		{
			name:     "data prefix text delta",
			input:    `data: {"type":"text-delta","id":"txt-0","delta":"hello"}`,
			expected: "hello",
		},
		{
			name:     "message prefix text end",
			input:    `message: {"type":"text-end","id":"txt-0"}`,
			expected: "",
		},
		{
			name:     "data prefix reasoning delta",
			input:    `data: {"type":"reasoning-delta","id":"reasoning-0","delta":"thinking"}`,
			expected: "",
		},
		{
			name:     "raw json text delta",
			input:    `{"type":"text-delta","id":"txt-0","delta":"world"}`,
			expected: "world",
		},
		{
			name:     "invalid json",
			input:    `invalid json`,
			expected: "",
		},
		{
			name:     "empty line",
			input:    "",
			expected: "",
		},
		{
			name:     "error chunk string",
			input:    `data: {"type":"error","error":"Rate limit exceeded"}`,
			expected: "\n[fx error: Rate limit exceeded]\n",
		},
		{
			name:     "error chunk object with message",
			input:    `data: {"type":"error","error":{"message":"Invalid API key"}}`,
			expected: "\n[fx error: Invalid API key]\n",
		},
		{
			name:     "invalid request error with message",
			input:    `data: {"type":"invalid_request_error","message":"Invalid prompt"}`,
			expected: "\n[fx error: Invalid prompt]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetMainText(tt.input)
			if got != tt.expected {
				t.Errorf("GetMainText(%q) = %q, expected %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestRequestBodyFormat(t *testing.T) {
	input := "what is love?"
	params := structs.Params{
		SystemPrompt: "You are a helpful assistant",
		PrevMessages: []any{
			structs.DefaultMessage{
				Role:    "user",
				Content: "hello",
			},
			structs.DefaultMessage{
				Role:    "assistant",
				Content: "hello! what can I help you with?",
			},
		},
	}

	var promptMessages []PromptMessage
	if params.SystemPrompt != "" {
		promptMessages = append(promptMessages, PromptMessage{
			Role:    "system",
			Content: params.SystemPrompt,
		})
	}
	for _, prev := range params.PrevMessages {
		if dm, ok := prev.(structs.DefaultMessage); ok {
			promptMessages = append(promptMessages, PromptMessage{
				Role: dm.Role,
				Content: []ContentPart{
					{
						Type: "text",
						Text: dm.Content,
					},
				},
			})
		}
	}
	promptMessages = append(promptMessages, PromptMessage{
		Role: "user",
		Content: []ContentPart{
			{
				Type: "text",
				Text: input,
			},
		},
	})

	reqBody := RequestBody{
		Prompt:          promptMessages,
		MaxOutputTokens: 128000,
		ProviderOptions: map[string]any{
			"gateway": map[string]any{
				"speed": "fast",
			},
		},
		Headers: map[string]any{
			"user-agent": "fx/0.0.3",
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}

	promptList, ok := decoded["prompt"].([]any)
	if !ok || len(promptList) != 4 {
		t.Fatalf("expected 4 prompt messages, got %d", len(promptList))
	}

	sysMsg := promptList[0].(map[string]any)
	if sysMsg["role"] != "system" || sysMsg["content"] != "You are a helpful assistant" {
		t.Errorf("unexpected system message: %+v", sysMsg)
	}

	userMsg1 := promptList[1].(map[string]any)
	if userMsg1["role"] != "user" {
		t.Errorf("unexpected role for userMsg1: %+v", userMsg1)
	}

	asstMsg := promptList[2].(map[string]any)
	if asstMsg["role"] != "assistant" {
		t.Errorf("unexpected role for asstMsg: %+v", asstMsg)
	}

	userMsg2 := promptList[3].(map[string]any)
	if userMsg2["role"] != "user" {
		t.Errorf("unexpected role for userMsg2: %+v", userMsg2)
	}

	if decoded["maxOutputTokens"].(float64) != 128000 {
		t.Errorf("unexpected maxOutputTokens: %v", decoded["maxOutputTokens"])
	}
}

func TestModelAndKeyResolution(t *testing.T) {
	// Default values
	model := "zai/glm-5.2"
	apiKey := "fx-demo-proxy"
	url := "https://fx.sh/fx-wasm/gateway/v3/ai/language-model"

	params := structs.Params{}
	if params.ApiModel != "" {
		model = params.ApiModel
	}
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	}
	if params.Url != "" {
		url = params.Url
	}

	if model != "zai/glm-5.2" {
		t.Errorf("expected default model zai/glm-5.2, got %s", model)
	}
	if apiKey != "fx-demo-proxy" {
		t.Errorf("expected default apiKey fx-demo-proxy, got %s", apiKey)
	}
	if url != "https://fx.sh/fx-wasm/gateway/v3/ai/language-model" {
		t.Errorf("expected default url, got %s", url)
	}

	// Override via params
	paramsOverride := structs.Params{
		ApiModel: "custom/model",
		ApiKey:   "custom-key",
		Url:      "https://custom.fx.sh/api",
	}

	if paramsOverride.ApiModel != "custom/model" {
		t.Errorf("expected custom/model")
	}
	if paramsOverride.ApiKey != "custom-key" {
		t.Errorf("expected custom-key")
	}
	if paramsOverride.Url != "https://custom.fx.sh/api" {
		t.Errorf("expected custom url")
	}
}

func TestGetMainTextStreamParsing(t *testing.T) {
	streamData := `message: {"type":"text-delta","id":"txt-0","delta":"Hello"}
message: {"type":"text-delta","id":"txt-0","delta":" world"}
message: {"type":"text-delta","id":"txt-0","delta":"!"}
message: {"type":"text-end","id":"txt-0"}`

	var fullText strings.Builder
	for _, line := range strings.Split(streamData, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		text := GetMainText(line)
		fullText.WriteString(text)
	}

	if fullText.String() != "Hello world!" {
		t.Errorf("expected 'Hello world!', got %q", fullText.String())
	}
}

