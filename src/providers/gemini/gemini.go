package gemini

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

	model := "gemini-2.0-flash"
	if params.ApiModel == "" {
		params.ApiModel = model
	}

	if params.Url == "" {
		params.Url = "https://generativelanguage.googleapis.com/v1beta/models"
	}

	// sending api key as query param could be security concern.
	url := params.Url + "/" + model + ":streamGenerateContent?alt=sse&key=" + params.ApiKey

	safeInput, _ := json.Marshal(input)

	dataStr := fmt.Sprintf(`{
		"systemInstruction": {
			"parts":[{
				"text": "%s"
			}]
		},
		"contents": [
		  %v
		  { 
			"role": "user",
			"parts": [ { "text": %v }]
		  }
	]}`, params.SystemPrompt, params.PrevMessages, string(safeInput))

	data := strings.NewReader(dataStr)

	req, err := http.NewRequest("POST", url, data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}

	req.Header.Set("Content-Type", "application/json")

	return client.Do(req)
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if len(line) > 1 {
		obj = strings.Split(line, "data: ")[1]
	}

	var d geminiResponse
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if len(d.Candidates) > 0 {
		mainText = d.Candidates[0].Content.Parts[0].Text
		return mainText
	}
	return ""
}

func GetInputResponseJson(input []byte, response []byte) string {
	return fmt.Sprintf(`{
			"parts": [{ "text": %v  }],
			"role": "user"
		},{
			"parts": [{ "text": %v  }],
			"role": "model"
		},
		`, string(input), string(response))
}
