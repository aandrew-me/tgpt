package sky

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
	Messages []any `json:"messages"`
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	url := "https://api.sky.foresko.com/v1/create-chat-completion"

	requestInfo := RequestBody{
		Messages: []any{},
	}

	if len(params.SystemPrompt) > 0 {
		requestInfo.Messages = append(requestInfo.Messages, structs.DefaultMessage{
			Content: params.SystemPrompt,
			Role:    "system",
		})
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

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("accept-charset", "UTF-8")
	req.Header.Add("accept-encoding", "gzip")
	req.Header.Add("connection", "Keep-Alive")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("user-agent", "ktor-client")

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
