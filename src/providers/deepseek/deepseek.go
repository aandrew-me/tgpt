package deepseek

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
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	model := "deepseek-reasoner"

	if params.ApiModel != "" {
		model = params.ApiModel
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	}

	url := "https://api.deepseek.com/chat/completions"

	requestInfo := RequestBody{
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
	var obj = "{}"
	if len(line) > 1 {
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
