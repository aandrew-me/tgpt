package openrouter

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
	defaultModel = "openrouter/free"
	defaultURL   = "https://openrouter.ai/api/v1/chat/completions"
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
	if value := os.Getenv("OPENROUTER_MODEL"); value != "" {
		return value
	}
	return defaultModel
}

func apiKey(params structs.Params) string {
	if params.ApiKey != "" {
		return params.ApiKey
	}
	if value := os.Getenv("OPENROUTER_API_KEY"); value != "" {
		return value
	}
	return os.Getenv("AI_API_KEY")
}

func endpoint(params structs.Params) string {
	if params.Url != "" {
		return params.Url
	}
	if value := os.Getenv("OPENROUTER_URL"); value != "" {
		return value
	}
	if value := os.Getenv("OPENROUTER_BASE_URL"); value != "" {
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
		return nil, fmt.Errorf("create OpenRouter request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if key := apiKey(params); key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}

	return httpClient.Do(req)
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if after, ok := strings.CutPrefix(line, "data: "); ok {
		obj = after
	}

	var d structs.CommonResponse
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if len(d.Choices) > 0 {
		mainText = d.Choices[0].Delta.Content
		return mainText
	}
	return ""
}
