package opengpts

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
)

type Message struct {
	Content string `json:"content"`
}

type Response struct {
	Messages []Message `json:"messages"`
}

func RandomString(length int) string {
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = characters[rand.Intn(len(characters))]
	}
	return string(result)
}

func NewRequest(input string, params structs.Params, prevMessages string) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
	safeInput, _ := json.Marshal(input)

	uuid := RandomString(36)

	var data = strings.NewReader(fmt.Sprintf(`{
	"input": {
		"messages": [
			%v
			{
				"content": %v,
				"additional_kwargs": {},
				"type": "human",
				"example": false
			}
		]
	},
	"assistant_id": "d50a5d6c-2598-437b-940e-e6918d19810c",
	"thread_id": ""
}
	`, prevMessages, string(safeInput)))

	req, err := http.NewRequest("POST", "https://opengpts-example-vz4y4ooboq-uc.a.run.app/runs/stream", data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Add("authority", "opengpts-example-vz4y4ooboq-uc.a.run.app")
	req.Header.Add("accept", "text/event-stream")
	req.Header.Add("accept-language", "en-US,en;q=0.7")
	req.Header.Add("cache-control", "no-cache")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("cookie", "opengpts_user_id="+uuid)
	req.Header.Add("origin", "https://opengpts-example-vz4y4ooboq-uc.a.run.app")
	req.Header.Add("pragma", "no-cache")
	req.Header.Add("referer", "https://opengpts-example-vz4y4ooboq-uc.a.run.app/")
	req.Header.Add("sec-fetch-site", "same-origin")
	req.Header.Add("sec-gpc", "1")
	req.Header.Add("user-agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if len(line) > 1 && strings.Contains(line, "data:") {
		obj = strings.Split(line, "data: ")[1]
	}

	var d Response
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if d.Messages != nil {
		mainText = d.Messages[len(d.Messages)-1].Content
		return mainText
	}
	return ""
}
