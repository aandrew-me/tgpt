package phind

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

	model := "Phind-70B"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	// preprompt := "You are a helpful assistant"

	// if params.Preprompt != "" {
	// 	preprompt = params.Preprompt
	// }

	// finalPreprompt := fmt.Sprintf(`
	// {
	// 	"content": "%v",
	// 	"role": "system"
	// },
	// `, preprompt)

	safeInput, _ := json.Marshal(input)

	var data = strings.NewReader(fmt.Sprintf(`{
		"additional_extension_context": "",
		"allow_magic_buttons": true,
		"is_vscode_extension": true,
		"message_history": [
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
		"requested_model": "%v",
		"user_input": %v
	}
	`, params.SystemPrompt, params.PrevMessages, string(safeInput), model, string(safeInput)))

	req, err := http.NewRequest("POST", "https://https.extension.phind.com/agent/", data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
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
