package opencode

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
	"github.com/aandrew-me/tgpt/v2/src/utils"
)

var defaultTools = []any{
	map[string]any{"type": "function", "function": map[string]any{"name": "bash"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "edit"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "glob"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "grep"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "question"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "read"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "skill"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "task"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "todowrite"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "webfetch"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "websearch"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "write"}},
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type RequestBody struct {
	Model         string         `json:"model"`
	Stream        bool           `json:"stream"`
	Messages      []any          `json:"messages"`
	Tools         []any          `json:"tools,omitempty"`
	ToolChoice    string         `json:"tool_choice,omitempty"`
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	model := "big-pickle"
	if params.ApiModel != "" {
		model = params.ApiModel
	} else if envModel := os.Getenv("OPENCODE_MODEL"); envModel != "" {
		model = envModel
	}

	apiKey := "public"
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	} else if envKey := os.Getenv("OPENCODE_API_KEY"); envKey != "" {
		apiKey = envKey
	} else if envKey := os.Getenv("AI_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	url := params.Url
	if url == "" {
		if envUrl := os.Getenv("OPENCODE_URL"); envUrl != "" {
			url = envUrl + "/v1/chat/completions"
		}
	}

	if url == "" {
		url = "https://opencode.ai/zen/v1/chat/completions"
	}

	messages := make([]any, 0, len(params.PrevMessages)+2)
	if params.SystemPrompt != "" {
		messages = append(messages, structs.DefaultMessage{
			Content: params.SystemPrompt,
			Role:    "system",
		})
	}

	tools := make([]any, 0, len(params.Tools)+len(defaultTools))
	seenTools := make(map[string]bool)

	for _, t := range params.Tools {
		if name := toolName(t); name != "" {
			seenTools[name] = true
		}
		tools = append(tools, t)
	}

	for _, dt := range defaultTools {
		if name := toolName(dt); name != "" && !seenTools[name] {
			seenTools[name] = true
			tools = append(tools, dt)
		}
	}

	toolChoice := "none"
	if len(params.Tools) > 0 {
		toolChoice = "auto"
	}

	requestInfo := RequestBody{
		Model:         model,
		Stream:        true,
		Messages:      messages,
		Tools:         tools,
		ToolChoice:    toolChoice,
		StreamOptions: &StreamOptions{IncludeUsage: true},
	}

	if len(params.PrevMessages) > 0 {
		requestInfo.Messages = append(requestInfo.Messages, params.PrevMessages...)
	}

	if input != "" {
		requestInfo.Messages = append(requestInfo.Messages, structs.DefaultMessage{
			Role:    "user",
			Content: input,
		})
	}

	jsonRequest, err := json.Marshal(requestInfo)

	if err != nil {
		log.Fatal("Failed to build user request")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonRequest))

	if err != nil {
		log.Fatal("Some error has occured.\nError:", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "*")
	req.Header.Set("Sec-Fetch-Mode", "cors")

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	randID, _ := utils.GenerateHexString(40)
	requestID, _ := utils.GenerateHexString(26)
	sesID, _ := utils.GenerateHexString(26)

	req.Header.Set("x-opencode-client", "desktop")
	req.Header.Set("x-opencode-project", randID)
	req.Header.Set("User-Agent", "opencode/1.18.18 ai-sdk/provider-utils/4.0.23 runtime/node.js/24")
	req.Header.Set("x-opencode-request", "msg_"+requestID)
	req.Header.Set("x-opencode-session", "ses_"+sesID)

	return client.Do(req)
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if strings.Contains(line, "data: ") {
		obj = strings.Split(line, "data: ")[1]
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

func toolName(t any) string {
	switch v := t.(type) {
	case map[string]any:
		if fn, ok := v["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok {
				return name
			}
		}
	}
	b, err := json.Marshal(t)
	if err == nil {
		var s struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if json.Unmarshal(b, &s) == nil {
			return s.Function.Name
		}
	}
	return ""
}

