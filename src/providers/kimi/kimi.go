package kimi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
)

type Extend struct {
	Sidebar bool `json:"sidebar"`
}

type ChatRequestBody struct {
	KimiPlusID        string `json:"kimiplus_id"`
	Extend            Extend `json:"extend"`
	Model             string `json:"model"`
	UseSearch         bool   `json:"use_search"`
	Messages          []any  `json:"messages"`
	Refs              []any  `json:"refs"`
	History           []any  `json:"history"`
	SceneLabels       []any  `json:"scene_labels"`
	UseSemanticMemory bool   `json:"use_semantic_memory"`
	UseDeepResearch   bool   `json:"use_deep_research"`
}

type RegisterResponseBody struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ChatIdResponse struct {
	Id string `json:"id"`
}

var deviceID = utils.GenerateRandomNumber(19)
var trafficID = utils.GenerateRandomNumber(19)

var chatId = ""
var accessToken = ""

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	if chatId == "" {
		accessToken = getAccessToken()
		chatId = getChatID(accessToken)
	}

	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	// Available: k2, k1.5
	model := "k2"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	requestInfo := ChatRequestBody{
		KimiPlusID: "kimi",
		Extend: Extend{
			Sidebar: true,
		},
		Model:             model,
		UseSearch:         true,
		Refs:              []any{},
		History:           []any{},
		SceneLabels:       []any{},
		UseSemanticMemory: false,
		UseDeepResearch:   false,
	}

	if params.ApiModel != "" {
		requestInfo.Model = params.ApiModel
	}

	mainMessage := structs.DefaultMessage{
		Role:    "user",
		Content: input,
	}

	messages := []any{}

	if len(params.SystemPrompt) > 0 {
		requestInfo.Messages = append(requestInfo.Messages, structs.DefaultMessage{
			Content: params.SystemPrompt,
			Role:    "system",
		})
	}

	messages = append(messages, mainMessage)

	requestInfo.Messages = messages

	jsonRequest, err := json.Marshal(requestInfo)

	if err != nil {
		log.Fatal("Failed to build user request")

	}

	apiUrl := "https://www.kimi.com/api/chat/" + chatId + "/completion/stream"

	req, err := http.NewRequest("POST", apiUrl, bytes.NewBuffer(jsonRequest))

	if err != nil {
		log.Fatal("Some error has occured.\nError:", err)
	}
	// Setting all the required headers
	req.Header.Add("accept", "application/json, text/plain, */*")
	req.Header.Add("accept-language", "en-US,en;q=0.9")
	req.Header.Add("authorization", "Bearer "+accessToken)
	req.Header.Add("content-type", "application/json")
	req.Header.Add("Cookie", "kimi-auth="+accessToken)
	req.Header.Add("origin", "https://www.kimi.com")
	req.Header.Add("priority", "u=1, i")
	req.Header.Add("referer", "https://www.kimi.com/chat/"+chatId)
	req.Header.Add("user-agent", "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/133.0")
	req.Header.Add("x-language", "en-US")
	req.Header.Add("x-msh-device-id", deviceID)
	req.Header.Add("x-msh-platform", "web")
	req.Header.Add("x-traffic-id", trafficID)

	// Return response
	return (client.Do(req))
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if len(line) > 1 {
		obj = strings.Split(line, "data: ")[1]
	}

	var d structs.KimiResponse
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if d.Event == "cmpl" {
		mainText = d.Text

		return mainText
	}

	return ""
}

func getAccessToken() string {
	url := "https://www.kimi.com/api/device/register"

	payload := strings.NewReader("{}")

	http_client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	req, _ := http.NewRequest("POST", url, payload)

	deviceID := utils.GenerateRandomNumber(19)
	trafficID := utils.GenerateRandomNumber(19)

	req.Header.Add("accept", "application/json, text/plain, */*")
	req.Header.Add("accept-language", "en-US,en;q=0.8")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("origin", "https://www.kimi.com")
	req.Header.Add("priority", "u=1, i")
	req.Header.Add("referer", "https://www.kimi.com/")
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-origin")
	req.Header.Add("sec-gpc", "1")
	req.Header.Add("user-agent", "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/133.0")
	req.Header.Add("x-language", "en-US")
	req.Header.Add("x-msh-device-id", deviceID)
	req.Header.Add("x-msh-platform", "web")
	req.Header.Add("x-traffic-id", trafficID)

	res, _ := http_client.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	jsonBody := string(body)

	var parsedBody RegisterResponseBody

	if err := json.Unmarshal([]byte(jsonBody), &parsedBody); err != nil {
		return ""
	}

	return parsedBody.AccessToken
}

func getChatID(accessToken string) string {
	url := "https://www.kimi.com/api/chat"

	payload := strings.NewReader(`
	{
	"name": "Unnamed Chat",
	"born_from": "home",
	"kimiplus_id": "kimi",
	"is_example": false,
	"source": "web",
	"tags": []
}
	`)

	http_client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Add("accept", "application/json, text/plain, */*")
	req.Header.Add("accept-language", "en-US,en;q=0.8")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("origin", "https://www.kimi.com")
	req.Header.Add("priority", "u=1, i")
	req.Header.Add("referer", "https://www.kimi.com/")
	req.Header.Add("sec-fetch-dest", "empty")
	req.Header.Add("sec-fetch-mode", "cors")
	req.Header.Add("sec-fetch-site", "same-origin")
	req.Header.Add("sec-gpc", "1")
	req.Header.Add("user-agent", "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/133.0")
	req.Header.Add("x-language", "en-US")
	req.Header.Add("x-msh-device-id", deviceID)
	req.Header.Add("x-msh-platform", "web")
	req.Header.Add("x-traffic-id", trafficID)
	req.Header.Add("Cookie", "kimi-auth="+accessToken)
	req.Header.Add("Authorization", "Bearer "+accessToken)

	res, _ := http_client.Do(req)

	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)

	jsonBody := string(body)

	var parsedBody ChatIdResponse

	if err := json.Unmarshal([]byte(jsonBody), &parsedBody); err != nil {
		return ""
	}

	return parsedBody.Id
}
