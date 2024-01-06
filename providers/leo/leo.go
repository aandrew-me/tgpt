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

	safeTxt, _ := json.Marshal(input)
	safeInput := fmt.Sprintf(`"<s>[INST] <<SYS>>\n\nYour name is Leo, a helpful, respectful and honest AI assistant created by the company Brave. You will be replying to a user of the Brave browser. Always respond in a neutral tone. Be polite and courteous. Answer concisely in no more than 50-80 words.\n\nPlease ensure that your responses are socially unbiased and positive in nature. If a question does not make any sense, or is not factually coherent, explain why instead of answering something not correct. If you don't know the answer to a question, please don't share false information.\n<</SYS>>\n\n%v [/INST] Here is your response:"`, string(safeTxt)[1:len(safeTxt)-1])

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
	req.Header.Set("accept", "text/event-stream")
	req.Header.Set("accept-language", "en-US,en;q=0.9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Return response
	return (client.Do(req))
}
