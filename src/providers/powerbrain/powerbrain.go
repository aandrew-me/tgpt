package powerbrain

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"os"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

type RequestBody struct {
	Model       string `json:"model"`
	Messages    []any  `json:"messages"`
	SecretToken string `json:"secret_token"`
	Action      string `json:"action"`
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	requestInfo := RequestBody{
		Model:       "gpt-5",
		SecretToken: "AIChatPowerBrain123@2024",
		Action:      "send_message",
	}

	if params.ApiModel != "" {
		requestInfo.Model = params.ApiModel
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

	apiUrl := "https://powerbrainai.com/app/backend/api/api.php"

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonRequest))

	if err != nil {
		log.Fatal("Some error has occured.\nError:", err)
	}
	// Setting all the required headers
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("User-Agent", "Dart/3.5 (dart:io)")

	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if len(line) > 1 {
		obj = line
	}

	var d structs.PowerBrainResponse
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if len(d.Data) > 0 {
		mainText = d.Data

		return mainText
	}
	return ""
}