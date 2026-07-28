package aihorde

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

type TextGenParams struct {
	MaxLength  int     `json:"max_length,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopP       float64 `json:"top_p,omitempty"`
	RepPen     float64 `json:"rep_pen,omitempty"`
}

type TextSubmitRequest struct {
	Prompt string         `json:"prompt"`
	Models []string       `json:"models"`
	Params *TextGenParams `json:"params,omitempty"`
}

type TextSubmitResponse struct {
	ID string `json:"id"`
}

type TextStatusResponse struct {
	Done      bool `json:"done"`
	Faulted   bool `json:"faulted"`
	Generations []struct {
		Text  string `json:"text"`
		State string `json:"state"`
		Model string `json:"model"`
	} `json:"generations"`
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	c, err := client.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	model := "koboldcpp/L3-8B-Stheno-v3.2"
	if params.ApiModel != "" {
		model = params.ApiModel
	} else if envModel := os.Getenv("AIHORDE_MODEL"); envModel != "" {
		model = envModel
	}

	apiKey := "0000000000"
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	} else if envKey := os.Getenv("AIHORDE_API_KEY"); envKey != "" {
		apiKey = envKey
	} else if envKey := os.Getenv("AI_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	// Build prompt with chat format
	var prompt string
	if params.SystemPrompt != "" {
		prompt = fmt.Sprintf("<|im_start|>system\n%s<|im_end|>\n<|im_start|>user\n%s<|im_end|>\n<|im_start|>assistant\n", params.SystemPrompt, input)
	} else {
		prompt = fmt.Sprintf("<|im_start|>user\n%s<|im_end|>\n<|im_start|>assistant\n", input)
	}

	// Add conversation history if present
	if len(params.PrevMessages) > 0 {
		history := ""
		for _, msg := range params.PrevMessages {
			if m, ok := msg.(structs.DefaultMessage); ok {
				if m.Content != "" {
					history += fmt.Sprintf("<|im_start|>%s\n%s<|im_end|>\n", m.Role, m.Content)
				}
			}
		}
		prompt = history + prompt
	}

	// Parse user-configured generation params with safe defaults
	maxLength := 200
	if v, err := strconv.Atoi(params.Max_length); err == nil && v > 0 {
		maxLength = v
	}

	temperature := 0.7
	if v, err := strconv.ParseFloat(params.Temperature, 64); err == nil && v > 0 && v <= 2 {
		temperature = v
	}

	topP := 0.9
	if v, err := strconv.ParseFloat(params.Top_p, 64); err == nil && v > 0 && v <= 1 {
		topP = v
	}

	submitReq := TextSubmitRequest{
		Prompt: prompt,
		Models: []string{model},
		Params: &TextGenParams{
			MaxLength:   maxLength,
			Temperature: temperature,
			TopP:        topP,
			RepPen:     1.1,
		},
	}

	jsonReq, err := json.Marshal(submitReq)
	if err != nil {
		return nil, fmt.Errorf("failed to build request: %w", err)
	}

	// Submit async text generation
	req, err := http.NewRequest("POST", "https://stablehorde.net/api/v2/generate/text/async", bytes.NewBuffer(jsonReq))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Client-Agent", "tgpt:v2:")

	res, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if res.StatusCode != 202 {
		return nil, fmt.Errorf("submission failed (HTTP %d): %s", res.StatusCode, string(body))
	}

	var submitResp TextSubmitResponse
	if err := json.Unmarshal(body, &submitResp); err != nil {
		return nil, fmt.Errorf("failed to parse submission response: %w", err)
	}

	// Poll for status (max 60 attempts × 2s = ~2 min)
	const maxAttempts = 60
	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(2 * time.Second)

		req, err := http.NewRequest("GET", "https://stablehorde.net/api/v2/generate/text/status/"+submitResp.ID, nil)
		if err != nil {
			return nil, fmt.Errorf("error checking status: %w", err)
		}
		req.Header.Set("apikey", apiKey)
		req.Header.Set("Client-Agent", "tgpt:v2:")

		res, err := c.Do(req)
		if err != nil {
			return nil, fmt.Errorf("error checking status: %w", err)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("failed to read status: %w", err)
		}

		var status TextStatusResponse
		if err := json.Unmarshal(body, &status); err != nil {
			return nil, fmt.Errorf("failed to parse status: %w", err)
		}

		if status.Faulted {
			return nil, fmt.Errorf("text generation failed")
		}

		if status.Done && len(status.Generations) > 0 {
			text := status.Generations[0].Text

			// Return as a simple HTTP response that GetMainText can parse
			respBody := io.NopCloser(strings.NewReader(text))
			return &http.Response{
				StatusCode: 200,
				Body:       respBody,
			}, nil
		}
	}

	return nil, fmt.Errorf("timed out waiting for text generation (~2 min)")
}

func GetMainText(line string) (mainText string) {
	if line != "" {
		return line
	}
	return ""
}
