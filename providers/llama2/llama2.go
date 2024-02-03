package llama2

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
)

func NewRequest(input string, params structs.Params, prevMessages string) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	model := "meta/llama-2-70b-chat"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	temperature := "0.75"
	if params.Temperature != "" {
		temperature = params.Temperature
	}

	top_p := "0.9"
	if params.Top_p != "" {
		top_p = params.Top_p
	}

	max_tokens := "800"
	if params.Max_length != "" {
		max_tokens = params.Max_length
	}

	safeInput, _ := json.Marshal(input)
	finalInput := string(safeInput)[1: len(string(safeInput)) - 1]

	prompt := fmt.Sprintf(`%v<s>[INST] %v [/INST]`, prevMessages, finalInput)

	var data = strings.NewReader(fmt.Sprintf(`
	{
		"prompt": "%v",
		"model": "%v",
		"systemPrompt": "You are a helpful assistant.",
		"temperature": %v,
		"topP": %v,
		"maxTokens": %v,
		"image": null,
		"audio": null
	}
	`, prompt, model, temperature, top_p, max_tokens))

	req, err := http.NewRequest("POST", "https://www.llama2.ai/api", data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Referer", "https://www.llama2.ai/")
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Origin", "https://www.llama2.ai")
	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	return line + "\n"
}
