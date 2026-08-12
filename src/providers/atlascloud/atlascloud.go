package atlascloud

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

const (
	defaultModel = "qwen/qwen3.8-max"
	defaultURL   = "https://api.atlascloud.ai/v1/chat/completions"
)

type RequestBody struct {
	Model    string `json:"model"`
	Stream   bool   `json:"stream"`
	Messages []any  `json:"messages"`
	Tools    []any  `json:"tools,omitempty"`
}

func model(params structs.Params) string {
	if params.ApiModel != "" {
		return params.ApiModel
	}
	if value := os.Getenv("ATLASCLOUD_MODEL"); value != "" {
		return value
	}
	return defaultModel
}

func apiKey(params structs.Params) string {
	if params.ApiKey != "" {
		return params.ApiKey
	}
	if value := os.Getenv("ATLASCLOUD_API_KEY"); value != "" {
		return value
	}
	return os.Getenv("AI_API_KEY")
}

func endpoint(params structs.Params) string {
	if params.Url != "" {
		return params.Url
	}
	if value := os.Getenv("ATLASCLOUD_URL"); value != "" {
		return value
	}
	if value := os.Getenv("ATLASCLOUD_BASE_URL"); value != "" {
		return strings.TrimSuffix(value, "/") + "/chat/completions"
	}
	return defaultURL
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	httpClient, err := client.NewClient()
	if err != nil {
		return nil, fmt.Errorf("create client: %w", err)
	}

	messages := make([]any, 0, len(params.PrevMessages)+2)
	if params.SystemPrompt != "" {
		messages = append(messages, structs.DefaultMessage{
			Content: params.SystemPrompt,
			Role:    "system",
		})
	}
	messages = append(messages, params.PrevMessages...)
	if input != "" {
		messages = append(messages, structs.DefaultMessage{
			Role:    "user",
			Content: input,
		})
	}

	requestInfo := RequestBody{
		Model:    model(params),
		Stream:   true,
		Messages: messages,
		Tools:    params.Tools,
	}
	jsonRequest, err := json.Marshal(requestInfo)
	if err != nil {
		log.Fatal("Failed to build user request")
	}

	req, err := http.NewRequest("POST", endpoint(params), bytes.NewBuffer(jsonRequest))
	if err != nil {
		return nil, fmt.Errorf("create Atlas Cloud request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := apiKey(params); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	return httpClient.Do(req)
}

func GetMainText(line string) string {
	obj, ok := strings.CutPrefix(line, "data: ")
	if !ok || obj == "[DONE]" {
		return ""
	}

	var response structs.CommonResponse
	if err := json.Unmarshal([]byte(obj), &response); err != nil {
		return ""
	}
	if len(response.Choices) == 0 {
		return ""
	}
	return response.Choices[0].Delta.Content
}
