package openai

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
)

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	model := "gpt-3.5-turbo"
	if params.ApiModel != "" {
		model = params.ApiModel
	} else if envModel := os.Getenv("OPENAI_MODEL"); envModel != "" {
		model = envModel
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	}

	url := params.Url
	if os.Getenv("OPENAI_URL") != "" {
		url = os.Getenv("OPENAI_URL")
	}

	temperature := "0.5"
	if params.Temperature != "" {
		temperature = params.Temperature
	}

	top_p := "0.5"
	if params.Top_p != "" {
		top_p = params.Top_p
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
		"temperature": %v,
		"top_p": %v
	}
	`, params.PrevMessages, string(safeInput), model, temperature, top_p))

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

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

	if d.Choices != nil && len(d.Choices) > 0 {
		mainText = d.Choices[0].Delta.Content
		return mainText
	}
	return ""
}
