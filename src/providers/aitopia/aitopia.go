package aitopia

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

const (
	defaultModel = "AITOPIA"
	defaultURL   = "https://extensions.aitopia.ai/ai/send"
)

type extraData struct {
	PromptMode bool `json:"prompt_mode"`
}

type historyItem struct {
	Item         string    `json:"item"`
	Role         string    `json:"role"`
	Model        string    `json:"model"`
	Title        any       `json:"title"`
	Loading      bool      `json:"loading"`
	ExtraData    extraData `json:"extra_data"`
	FinishReason string    `json:"finish_reason,omitempty"`
}

type languageDetail struct {
	LangCode string `json:"lang_code"`
	Name     string `json:"name"`
	Title    string `json:"title"`
}

type requestBody struct {
	History        []historyItem  `json:"history"`
	Text           string         `json:"text"`
	Model          string         `json:"model"`
	Stream         bool           `json:"stream"`
	Mode           string         `json:"mode"`
	PromptMode     bool           `json:"prompt_mode"`
	ExtraKey       string         `json:"extra_key"`
	ExtraData      extraData      `json:"extra_data"`
	LanguageDetail languageDetail `json:"language_detail"`
	IsContinue     bool           `json:"is_continue"`
	LangCode       string         `json:"lang_code"`
}

func generateHopeKey() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	httpClient, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	model := defaultModel
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	url := defaultURL
	if params.Url != "" {
		url = params.Url
	}

	history := []historyItem{}
	if len(params.PrevMessages) > 0 {
		for _, prev := range params.PrevMessages {
			var role, content string
			switch msg := prev.(type) {
			case structs.DefaultMessage:
				role, content = msg.Role, msg.Content
			case map[string]any:
				role, _ = msg["role"].(string)
				content, _ = msg["content"].(string)
			default:
				b, _ := json.Marshal(prev)
				var dm structs.DefaultMessage
				if json.Unmarshal(b, &dm) == nil && dm.Role != "" {
					role, content = dm.Role, dm.Content
				}
			}
			if role == "" {
				continue
			}
			itemRole := role
			if role == "assistant" {
				itemRole = "system"
			}
			history = append(history, historyItem{
				Item:      content,
				Role:      itemRole,
				Model:     model,
				ExtraData: extraData{PromptMode: false},
			})
		}
	}

	history = append(history, historyItem{
		Item:      input,
		Role:      "user",
		Model:     model,
		ExtraData: extraData{PromptMode: false},
	})
	history = append(history, historyItem{
		Item:      "",
		Role:      "system",
		Model:     model,
		Loading:   true,
		ExtraData: extraData{PromptMode: false},
	})

	body := requestBody{
		History:  history,
		Text:     input,
		Model:    model,
		Stream:   true,
		Mode:     "ai_chat",
		ExtraKey: "__all",
		ExtraData: extraData{PromptMode: false},
		LanguageDetail: languageDetail{
			LangCode: "en",
			Name:     "English",
			Title:    "English",
		},
		LangCode: "en",
	}

	jsonRequest, err := json.Marshal(body)
	if err != nil {
		log.Fatal("Failed to build aitopia request")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonRequest))
	if err != nil {
		log.Fatal("Some error has occurred.\nError:", err)
	}

	hopeKey := generateHopeKey()

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/plain")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("hopekey", hopeKey)
	req.Header.Set("Origin", "chrome-extension://becfinhbfclcgokjlobojlnldbfillpf")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/151.0.0.0 Safari/537.36")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "none")

	return httpClient.Do(req)
}

type choiceEntry struct {
	Delta struct {
		Content string `json:"content"`
	} `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type streamChunk struct {
	Choices json.RawMessage `json:"choices"`
}

func GetMainText(line string) string {
	obj := strings.TrimPrefix(line, "data: ")
	if obj == "" || obj == "[DONE]" {
		return ""
	}

	var chunk streamChunk
	if err := json.Unmarshal([]byte(obj), &chunk); err != nil || len(chunk.Choices) == 0 {
		return ""
	}

	// Actual aitopia format: {"choices":{"0":{...},...}}
	if chunk.Choices[0] == '{' {
		var asMap map[string]choiceEntry
		if json.Unmarshal(chunk.Choices, &asMap) == nil {
			if entry, ok := asMap["0"]; ok {
				return entry.Delta.Content
			}
		}
		return ""
	}

	// Fallback: OpenAI-compatible array form {"choices":[{...},...]}
	var asSlice []choiceEntry
	if json.Unmarshal(chunk.Choices, &asSlice) == nil && len(asSlice) > 0 {
		return asSlice[0].Delta.Content
	}
	return ""
}
