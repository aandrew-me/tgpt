package isou

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/fatih/color"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

// NewRequest sends a chat request to the isou.chat API and returns the streaming HTTP response.
func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()

	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	model := "gpt-5.4-mini"

	if params.ApiModel != "" {
		model = params.ApiModel
	}

	var data = strings.NewReader(fmt.Sprintf(`{
		"model": "%v",
		"provider": "openai",
		"language": "all",
		"categories": [
			"general"
		],
		"engine": "SEARXNG",
		"messages": [{"role": "user", "content": "%s"}]
	}
	`, model, input))

	link := "https://isou.chat/api/chat"

	req, err := http.NewRequest("POST", link, data)

	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "text/event-stream")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Referer", "https://isou.chat/chat")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", "https://isou.chat")
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/148.0")

	// Return response
	return (client.Do(req))
}

// GetMainText parses a server-sent event line from the isou API and returns the text content or formatted source citations.
func GetMainText(line string) (mainText string) {
	var obj = "{}"

	if len(line) > 1 {
		parts := strings.SplitN(line, "data:", 2)
		if len(parts) > 1 {
			obj = parts[1]
		}
	}

	type Context struct {
		Id      int    `json:"id"`
		Name    string `json:"name"`
		Source  string `json:"url"`
	}

	type InnerData struct {
		Content  string    `json:"content"`
		Role     string    `json:"role"`
		Contexts []Context `json:"contexts"`
	}

	type OuterData struct {
		Data InnerData `json:"data"`
	}

	var outer OuterData

	if err := json.Unmarshal([]byte(obj), &outer); err != nil {
		return ""
	}

	inner := outer.Data

	yellow := color.New(color.FgHiYellow)

	if len(inner.Contexts) > 0 {
		for _, ctx := range inner.Contexts {
			mainText += yellow.Sprintf("%v. Name: %v, Source: %v\n", ctx.Id, ctx.Name, ctx.Source)
		}
		return mainText
	}

	if inner.Content != "" {
		mainText = inner.Content
		return mainText
	}

	return ""
}
