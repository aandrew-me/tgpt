package isou

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

	model := "deepseek-chat"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	apiKey := params.ApiKey
	url := params.Url

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
		"model": "%v",
		"messages": [
			{
				"role": "user",
				"content": %v
			}
		],
		"temperature": %v,
		"top_p": %v
	}
	`, model, string(safeInput), temperature, top_p))

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
