package deepseek

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	model := "deepseek-reasoner"
	if params.ApiModel != "" {
		model = params.ApiModel
	} else if envModel := os.Getenv("DEEPSEEK_MODEL"); envModel != "" {
		model = envModel
	}

	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	}

	temperature := "0.6"
	if params.Temperature != "" {
		temperature = params.Temperature
	}

	top_p := "1"
	if params.Top_p != "" {
		top_p = params.Top_p
	}

	safeInput, _ := json.Marshal(input)

	var data = strings.NewReader(fmt.Sprintf(`{
		"messages": [
			{
				"content": "%s",
				"role": "system"
			},
			%v
			{
				"content": %v,
				"role": "user"
			}
		],
		"model": "%v",
		"stream": true,
		"temperature": %v,
		"top_p": %v
	}
	`, params.SystemPrompt, params.PrevMessages, string(safeInput), model, temperature, top_p))

	req, err := http.NewRequest("POST", "https://api.deepseek.com/chat/completions", data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	return client.Do(req)
}

func GetMainText(line string) (mainText string) {
	// fmt.Println(line)
	var obj = "{}"
	if len(line) > 1 {
		objArr := strings.Split(line, "data: ")
		if len(objArr) > 1 {
			obj = objArr[1]
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
