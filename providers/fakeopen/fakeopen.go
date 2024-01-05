package fakeopen

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
)

type Response struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

func NewRequest(input string, params structs.Params, prevMessages string) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	model := "gpt-3.5-turbo"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	safeInput, _ := json.Marshal(input)

	var data = strings.NewReader(fmt.Sprintf(`{
		"frequency_penalty": 0,
		"messages": [
			%v
			{
				"content": %v,
				"role": "user"
			}
		],
		"model": "%v",
		"presence_penalty": 0,
		"stream": true,
		"temperature": 1,
		"top_p": 1
	}
	`, prevMessages, string(safeInput), model))

	req, err := http.NewRequest("POST", "https://ai.fakeopen.com/v1/chat/completions", data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", "Bearer pk-this-is-a-real-free-pool-token-for-everyone")
	req.Header.Set("Origin", "https://chat.geekgpt.org")
	req.Header.Set("Referrer", "https://chat.geekgpt.org/")

	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if len(line) > 1 {
		obj = strings.Split(line, "data: ")[1]
	}

	var d Response
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if d.Choices != nil {
		mainText = d.Choices[0].Delta.Content
		return mainText
	}
	return ""
}
