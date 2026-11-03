package pollinations

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
	Model    string `json:"model"`
	Stream   bool   `json:"stream"`
	Messages []any  `json:"messages"`
	Tools    []any  `json:"tools,omitempty"`

	Temperature *float64 `json:"temperature,omitempty"`
	TopP        *float64 `json:"top_p,omitempty"`
}

func newRequestBody(params structs.Params) RequestBody {
	body := RequestBody{
		Model:  "openai",
		Stream: true,
		Tools:  params.Tools,
	}
	body.Temperature, body.TopP = params.SamplingOptions()
	return body
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	requestInfo := newRequestBody(params)

	apiKey := params.ApiKey
	if apiKey == "" {
		apiKey = os.Getenv("POLLINATIONS_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}

	if params.ApiModel != "" {
		requestInfo.Model = params.ApiModel
	} else if envModel := os.Getenv("POLLINATIONS_MODEL"); envModel != "" {
		requestInfo.Model = envModel
	}

	messages := make([]any, 0, len(params.PrevMessages)+2)
	if params.SystemPrompt != "" {
		messages = append(messages, structs.DefaultMessage{
			Role:    "system",
			Content: params.SystemPrompt,
		})
	}

	if len(params.PrevMessages) > 0 {
		messages = append(messages, params.PrevMessages...)
	}

	if input != "" {
		messages = append(messages, structs.DefaultMessage{
			Role:    "user",
			Content: input,
		})
	}

	requestInfo.Messages = messages

	jsonRequest, err := json.Marshal(requestInfo)

	if err != nil {
		log.Fatal("Failed to build user request")

	}

	apiUrl := "https://text.pollinations.ai/openai"

	if apiKey != "" {
		apiUrl = "https://gen.pollinations.ai/v1/chat/completions"
	}

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonRequest))

	if err != nil {
		log.Fatal("Some error has occured.\nError:", err)
	}
	// Setting all the required headers
	req.Header.Add("Content-Type", "application/json")

	if apiKey != "" {
		req.Header.Add("Authorization", "Bearer "+apiKey)
	}

	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if after, ok := strings.CutPrefix(line, "data: "); ok {
		obj = after
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
