package ollama

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

func NewCloudRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	model := params.ApiModel
	if model == "" {
		model = os.Getenv("OLLAMA_MODEL")
	}
	if model == "" {
		model = "gpt-oss:120b"
	}

	apiKey := params.ApiKey
	if apiKey == "" {
		apiKey = os.Getenv("OLLAMA_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}

	requestInfo := struct {
		Model    string `json:"model"`
		Stream   bool   `json:"stream"`
		Messages []any  `json:"messages"`
	}{
		Model:  model,
		Stream: true,
		Messages: []any{
			structs.DefaultMessage{
				Content: params.SystemPrompt,
				Role:    "system",
			},
		},
	}

	if len(params.PrevMessages) > 0 {
		requestInfo.Messages = append(requestInfo.Messages, params.PrevMessages...)
	}

	requestInfo.Messages = append(requestInfo.Messages, structs.DefaultMessage{
		Role:    "user",
		Content: input,
	})

	jsonRequest, err := json.Marshal(requestInfo)

	if err != nil {
		log.Fatal("Failed to build user request")
	}

	req, err := http.NewRequest("POST", "https://ollama.com/v1/chat/completions", bytes.NewBuffer(jsonRequest))

	if err != nil {
		log.Fatal("Some error has occured.\nError:", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	return client.Do(req)
}

func GetCloudMainText(line string) (mainText string) {
	obj := "{}"
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
