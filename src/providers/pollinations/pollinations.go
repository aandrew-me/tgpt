package pollinations

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
	Model       string `json:"model"`
	Referrer    string `json:"referrer"`
	Stream      bool   `json:"stream"`
	Messages    []any  `json:"messages"`
	Temperature string `json:"temperature"`
	Top_p       string `json:"top_p"`
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	requestInfo := RequestBody{
		Model:       "openai",
		Stream:      true,
		Referrer:    "tgpt",
		Temperature: "1",
		Top_p:       "1",
	}

	apiKey := params.ApiKey

	if params.ApiModel != "" {
		requestInfo.Model = params.ApiModel
	}

	if params.Temperature != "" {
		requestInfo.Temperature = params.Temperature
	}

	if params.Top_p != "" {
		requestInfo.Top_p = params.Top_p
	}

	systemMessage := structs.DefaultMessage{
		Role:    "system",
		Content: params.SystemPrompt,
	}

	mainMessage := structs.DefaultMessage{
		Role:    "user",
		Content: input,
	}

	messages := []any{systemMessage}

	if len(params.PrevMessages) > 0 {
		messages = append(messages, params.PrevMessages...)
	}

	messages = append(messages, mainMessage)

	requestInfo.Messages = messages

	jsonRequest, err := json.Marshal(requestInfo)

	if err != nil {
		log.Fatal("Failed to build user request")

	}

	req, err := http.NewRequest("POST", "https://text.pollinations.ai/openai", bytes.NewBuffer(jsonRequest))

	if err != nil {
		log.Fatal("Some error has occured.\nError:", err)
	}
	// Setting all the required headers
	req.Header.Add("Content-Type", "application/json")

	if apiKey != "" {
		req.Header.Add("Authorization", "Bearer "+apiKey)
	}

	// Return response
	return (client.Do(req))
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
