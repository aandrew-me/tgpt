package isou

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"
	"github.com/fatih/color"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	model := "deepseek-ai/DeepSeek-R1-Distill-Qwen-32B"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	var data = strings.NewReader(fmt.Sprintf(`{
		"stream": true,
		"model": "%v",
		"provider": "siliconflow",
		"mode": "deep",
		"language": "all",
		"categories": [
			"science"
		],
		"engine": "SEARXNG",
		"locally": false,
		"reload": false
	}
	`, model))

	query := url.QueryEscape(input)
	link := fmt.Sprintf("https://isou.chat/api/search?q=%v", query)

	req, err := http.NewRequest("POST", link, data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Referer", "https://isou.chat/search")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", "https://isou.chat")
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0")

	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if len(line) > 1 {
		parts := strings.SplitN(line, "data:", 2)
		if len(parts) > 1 {
			obj = parts[1]
		}
	}

	type Context struct {
		Name string `json:"name"`
		Source string `json:"url"`
		Id int `json:"id"`
	}

	type InnerData struct {
		Content string `json:"content"`
		ReasoningContent string `json:"reasoningContent"`
		Context *Context `json:"context"`
	}

	type OuterData struct {
		Data string `json:"data"`
	}

	var outer OuterData
	if err := json.Unmarshal([]byte(obj), &outer); err != nil {
		return ""
	}

	var inner InnerData
	if err := json.Unmarshal([]byte(outer.Data), &inner); err != nil {
		return ""
	}

	italic := color.New(color.Italic)
	yellow := color.New(color.FgHiYellow)

	if inner.Context != nil {
		mainText := yellow.Sprintf("%v. Name: %v, Source: %v\n", inner.Context.Id, inner.Context.Name, inner.Context.Source)

		return mainText
	}

	if inner.ReasoningContent != "" {
		mainText = italic.Sprint(inner.ReasoningContent)
		
		return mainText
	}

	if inner.Content != "" {
		mainText = inner.Content
		return mainText
	}

	return ""
}
