package blackboxai

import (
	"encoding/json"
	"fmt"
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

	safeInput, _ := json.Marshal(input)

	var data = strings.NewReader(fmt.Sprintf(`
	{
		"messages": [
			%v
			{
				"content": %v,
				"role": "user"
			}
		],
		"id": "",
		"previewToken": null,
		"userId": "",
		"codeModelMode": true,
		"agentMode": {},
		"trendingAgentMode": {},
		"isMicMode": false
	}
	`,params.PrevMessages, string(safeInput)))

	req, err := http.NewRequest("POST", "https://www.blackbox.ai/api/chat", data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Add("Accept", "*/*")
	req.Header.Add("Accept-Language", "en-US,en;q=0.5")
	req.Header.Add("Referer", "https://www.blackbox.ai/")
	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Origin", "https://www.blackbox.ai")
	req.Header.Add("Alt-Used", "www.blackbox.ai")
	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	return line + "\n"
}
