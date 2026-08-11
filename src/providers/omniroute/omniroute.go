package omniroute

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

type RequestBody struct {
	Model    string `json:"model"`
	Stream   bool   `json:"stream"`
	Messages []any  `json:"messages"`
	Tools    []any  `json:"tools,omitempty"`
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	model := "auto"
	if params.ApiModel != "" {
		model = params.ApiModel
	} else if envModel := os.Getenv("OMNIROUTE_MODEL"); envModel != "" {
		model = envModel
	}

	apiKey := params.ApiKey
	if apiKey == "" {
		apiKey = os.Getenv("OMNIROUTE_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}

	url := params.Url
	if url == "" {
		url = os.Getenv("OMNIROUTE_URL")
	}
	if url == "" {
		if baseUrl := os.Getenv("OMNIROUTE_BASE_URL"); baseUrl != "" {
			url = strings.TrimSuffix(baseUrl, "/") + "/chat/completions"
		}
	}
	if url == "" {
		url = "http://localhost:20128/v1/chat/completions"
	}

	messages := make([]any, 0, len(params.PrevMessages)+2)
	if params.SystemPrompt != "" {
		messages = append(messages, structs.DefaultMessage{
			Content: params.SystemPrompt,
			Role:    "system",
		})
	}

	requestInfo := RequestBody{
		Model:    model,
		Stream:   true,
		Messages: messages,
		Tools:    params.Tools,
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

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	return client.Do(req)
}

func GetMainText(line string) (mainText string) {
	obj := "{}"
	if after, ok := strings.CutPrefix(line, "data: "); ok {
		obj = after
	}

	var d structs.CommonResponse
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if len(d.Choices) > 0 {
		return d.Choices[0].Delta.Content
	}
	return ""
}
