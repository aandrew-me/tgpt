package duckduckgo

import (
	"bufio"
	"encoding/json"
	"fmt"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestData struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

func NewRequest(input string, params structs.Params, prevMessages string) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	headers := map[string]string{
		"User-Agent":      "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0",
		"Accept":          "text/event-stream",
		"Accept-Language": "en-US;q=0.7,en;q=0.3",
		"Accept-Encoding": "gzip, deflate, br",
		"Referer":         "https://duckduckgo.com/",
		"Content-Type":    "application/json",
		"Origin":          "https://duckduckgo.com",
		"Connection":      "keep-alive",
		"Cookie":          "dcm=1",
		"Sec-Fetch-Dest":  "empty",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Site":  "same-origin",
		"Pragma":          "no-cache",
		"TE":              "trailers",
		"x-vqd-accept":    "1",
		"Cache-Control":   "no-store",
	}

	statusReq, err := http.NewRequest("GET", "https://duckduckgo.com/duckchat/v1/status", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating status request: %v", err)
	}

	for key, value := range headers {
		statusReq.Header.Set(key, value)
	}

	statusResp, err := client.Do(statusReq)
	if err != nil {
		return nil, fmt.Errorf("error making status request: %v", err)
	}
	defer statusResp.Body.Close()

	token := statusResp.Header.Get("x-vqd-4")
	headers["x-vqd-4"] = token

	model := "gpt-4o-mini"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	var messageHistory []Message
	if prevMessages != "" {
		err = json.Unmarshal([]byte(prevMessages), &messageHistory)
		if err != nil {
			return nil, fmt.Errorf("error unmarshalling prevMessages: %v", err)
		}
	}

	messageHistory = append(messageHistory, Message{
		Role:    "user",
		Content: input,
	})

	data := RequestData{
		Model:    model,
		Messages: messageHistory,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("error marshalling request data: %v", err)
	}

	req, err := http.NewRequest("POST", "https://duckduckgo.com/duckchat/v1/chat", strings.NewReader(string(jsonData)))
	if err != nil {
		return nil, fmt.Errorf("error creating chat request: %v", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making chat request: %v", err)
	}

	return resp, nil
}

func GetMainText(line string) (mainText string) {
	if len(line) > 6 && line[6] == '{' {
		var dat map[string]interface{}
		if err := json.Unmarshal([]byte(line[6:]), &dat); err == nil {
			if message, ok := dat["message"].(string); ok {
				return strings.ReplaceAll(message, "\\n", "\n")
			}
		}
	}
	return ""
}

func HandleResponse(resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			mainText := GetMainText(line)
			if mainText != "" {
				fmt.Print(mainText)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}
	return nil
}
