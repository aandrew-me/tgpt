package phind

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

type Message struct {
	Content string `json:"content"`
	Role    string `json:"role"`
}
type RequestBody struct {
	AdditionalExtensionContext string `json:"additional_extension_context"`
	AllowMagicButtons          bool   `json:"allow_magic_buttons"`
	IsVSCodeExtension          bool   `json:"is_vscode_extension"`
	MessageHistory             []any  `json:"message_history"`
	RequestedModel             string `json:"requested_model"`
	UserInput                  string `json:"user_input"`
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()

	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	model := "Phind-70B"

	if params.ApiModel != "" {
		model = params.ApiModel
	}

	requestInfo := RequestBody{
		AdditionalExtensionContext: "",
		AllowMagicButtons:          true,
		IsVSCodeExtension:          true,
		RequestedModel:             model,
		UserInput:                  input,
		MessageHistory: []any{
			structs.DefaultMessage{
				Content: params.SystemPrompt,
				Role:    "system",
			},
		},
	}

	if len(params.PrevMessages) > 0 {
		requestInfo.MessageHistory = append(requestInfo.MessageHistory, params.PrevMessages...)
	}

	requestInfo.MessageHistory = append(requestInfo.MessageHistory, structs.DefaultMessage{
		Role:    "user",
		Content: input,
	})

	jsonRequest, err := json.Marshal(requestInfo)

	if err != nil {
		log.Fatal("Failed to build user request")
	}

	req, err := http.NewRequest("POST", "https://https.extension.phind.com/agent/", bytes.NewBuffer(jsonRequest))

	if err != nil {
		log.Fatal("Some error has occured.\nError:", err)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "Identity")

	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"

	if len(line) > 1 {
		parts := strings.Split(line, "data: ")
		if len(parts) > 1 {
			obj = parts[1]
		}
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
