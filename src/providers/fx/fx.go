package fx

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

type ContentPart struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type PromptMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

type RequestBody struct {
	Prompt          []PromptMessage `json:"prompt"`
	MaxOutputTokens int             `json:"maxOutputTokens,omitempty"`
	ProviderOptions map[string]any  `json:"providerOptions,omitempty"`
	Headers         map[string]any  `json:"headers,omitempty"`
}

func generateSessionID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%d-%d000000-%s", time.Now().UnixMilli(), time.Now().UnixMilli(), hex.EncodeToString(b))
}

// NewRequest sends a prompt to the fx.sh gateway and returns the streaming response.
func NewRequest(input string, params structs.Params) (*http.Response, error) {
	httpClient, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	model := "zai/glm-5.2"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	apiKey := "fx-demo-proxy"
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	}

	url := params.Url
	if url == "" {
		url = "https://fx.sh/fx-wasm/gateway/v3/ai/language-model"
	}

	var promptMessages []PromptMessage

	if params.SystemPrompt != "" {
		promptMessages = append(promptMessages, PromptMessage{
			Role:    "system",
			Content: params.SystemPrompt,
		})
	}

	if len(params.PrevMessages) > 0 {
		for _, prev := range params.PrevMessages {
			switch msg := prev.(type) {
			case structs.DefaultMessage:
				if msg.Role == "system" {
					promptMessages = append(promptMessages, PromptMessage{
						Role:    "system",
						Content: msg.Content,
					})
				} else {
					promptMessages = append(promptMessages, PromptMessage{
						Role: msg.Role,
						Content: []ContentPart{
							{
								Type: "text",
								Text: msg.Content,
							},
						},
					})
				}
			case map[string]any:
				role, _ := msg["role"].(string)
				content, _ := msg["content"].(string)
				if role == "system" {
					promptMessages = append(promptMessages, PromptMessage{
						Role:    "system",
						Content: content,
					})
				} else {
					promptMessages = append(promptMessages, PromptMessage{
						Role: role,
						Content: []ContentPart{
							{
								Type: "text",
								Text: content,
							},
						},
					})
				}
			default:
				b, _ := json.Marshal(prev)
				var dm structs.DefaultMessage
				if json.Unmarshal(b, &dm) == nil && dm.Role != "" {
					if dm.Role == "system" {
						promptMessages = append(promptMessages, PromptMessage{
							Role:    "system",
							Content: dm.Content,
						})
					} else {
						promptMessages = append(promptMessages, PromptMessage{
							Role: dm.Role,
							Content: []ContentPart{
								{
									Type: "text",
									Text: dm.Content,
								},
							},
						})
					}
				}
			}
		}
	}

	if input != "" {
		promptMessages = append(promptMessages, PromptMessage{
			Role: "user",
			Content: []ContentPart{
				{
					Type: "text",
					Text: input,
				},
			},
		})
	}

	reqBody := RequestBody{
		Prompt:          promptMessages,
		MaxOutputTokens: 128000,
		ProviderOptions: map[string]any{
			"gateway": map[string]any{
				"speed": "fast",
			},
		},
		Headers: map[string]any{
			"user-agent": "fx/0.0.3",
		},
	}

	jsonRequest, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal("Failed to build user request")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonRequest))
	if err != nil {
		log.Fatal("Some error has occurred.\nError:", err)
	}

	sessionID := generateSessionID()

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("ai-gateway-protocol-version", "0.0.1")
	req.Header.Set("ai-language-model-id", model)
	req.Header.Set("ai-language-model-specification-version", "4")
	req.Header.Set("ai-language-model-streaming", "true")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("http-referer", "https://github.com/vercel-labs/fx")
	req.Header.Set("Origin", "https://fx.sh")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Referer", "https://fx.sh/")
	req.Header.Set("sec-fetch-dest", "empty")
	req.Header.Set("sec-fetch-mode", "cors")
	req.Header.Set("sec-fetch-site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0")
	req.Header.Set("x-session-affinity", sessionID)
	req.Header.Set("x-session-id", sessionID)
	req.Header.Set("x-title", "fx")

	return httpClient.Do(req)
}

type StreamChunk struct {
	Type    string `json:"type"`
	ID      string `json:"id,omitempty"`
	Delta   string `json:"delta,omitempty"`
	Text    string `json:"text,omitempty"`
	Error   any    `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
}

// GetMainText parses server-sent events from the fx.sh gateway stream.
func GetMainText(line string) string {
	obj := line
	if strings.HasPrefix(line, "message: ") {
		obj = strings.TrimPrefix(line, "message: ")
	} else if strings.HasPrefix(line, "data: ") {
		obj = strings.TrimPrefix(line, "data: ")
	} else if strings.Contains(line, "message:") {
		parts := strings.SplitN(line, "message:", 2)
		if len(parts) > 1 {
			obj = strings.TrimSpace(parts[1])
		}
	} else if strings.Contains(line, "data:") {
		parts := strings.SplitN(line, "data:", 2)
		if len(parts) > 1 {
			obj = strings.TrimSpace(parts[1])
		}
	}

	var chunk StreamChunk
	if err := json.Unmarshal([]byte(obj), &chunk); err != nil {
		return ""
	}

	if chunk.Type == "text-delta" {
		return chunk.Delta
	}

	if chunk.Type == "error" || chunk.Type == "invalid_request_error" || chunk.Error != nil {
		var errMsg string
		switch e := chunk.Error.(type) {
		case string:
			errMsg = e
		case map[string]any:
			if msg, ok := e["message"].(string); ok && msg != "" {
				errMsg = msg
			} else {
				b, _ := json.Marshal(e)
				errMsg = string(b)
			}
		default:
			if chunk.Message != "" {
				errMsg = chunk.Message
			} else if chunk.Error != nil {
				errMsg = fmt.Sprintf("%v", chunk.Error)
			}
		}
		if errMsg != "" {
			return fmt.Sprintf("\n[fx error: %s]\n", errMsg)
		}
	}

	return ""
}
