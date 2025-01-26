package isou

import (
	// "encoding/json"
	"encoding/json"
	"fmt"
	"net/url"
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

	var data = strings.NewReader(fmt.Sprintf(`{
		"stream": true,
		"model": "%v",
		"provider": "ollama",
		"mode": "deep",
		"language": "all",
		"categories": [
			"general"
		],
		"engine": "SEARXNG",
		"locally": false,
		"reload": false
	}
	`, model))

	query := url.QueryEscape(input);
	link := fmt.Sprintf("https://isou.chat/api/search?q=%v", query);

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

	type InnerData struct {
		Answer string `json:"answer"`
		
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

	if inner.Answer != "" {
		mainText = inner.Answer
		return mainText
	}

	return ""
}

