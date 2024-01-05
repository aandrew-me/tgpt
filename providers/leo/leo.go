package leo

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
	http "github.com/bogdanfinn/fhttp"
)

type Response struct {
	Completion string `json:"completion"`
}

func GetMainText(line string) string {
	var obj = "{}"
	if len(line) > 1 {
		obj = strings.Split(line, "data: ")[1]
	}

	var d Response
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if d.Completion != "" {
		mainText := d.Completion
		return mainText
	}
	return ""
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	model := "llama-2-13b-chat"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	key := "qztbjzBqJueQZLFkwTTJrieu8Vw3789u"
	if params.ApiKey != "" {
		key = params.ApiKey
	}

	safeInput, _ := json.Marshal("[INST] " + input + " [/INST]  ")

	var data = strings.NewReader(fmt.Sprintf(`{
		"max_tokens_to_sample": 600,
		"model": "%v",
		"prompt": %v,
		"stop_sequences": [
			"</response>",
			"</s>"
		],
		"stream": true,
		"temperature": 0.2,
		"top_k": -1,
		"top_p": 0.999
	}
	`, model, string(safeInput)))

	req, err := http.NewRequest("POST", "https://ai-chat.bsg.brave.com/v1/complete", data)
	if err != nil {
		fmt.Fprintln(os.Stderr, "\nSome error has occurred.")
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-brave-key", key)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:99.0) Gecko/20100101 Firefox/110.0")

	// Return response
	return (client.Do(req))
}
