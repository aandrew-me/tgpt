package aihorde

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
)

type GenerateRequest struct {
	Prompt string   `json:"prompt"`
	Models []string `json:"models"`
	Params *struct {
		Width  int `json:"width,omitempty"`
		Height int `json:"height,omitempty"`
		Steps  int `json:"steps,omitempty"`
	} `json:"params,omitempty"`
}

type GenerateResponse struct {
	ID string `json:"id"`
}

type StatusResponse struct {
	Done   bool `json:"done"`
	Faulted bool `json:"faulted"`
	Generations []struct {
		Img string `json:"img"`
	} `json:"generations"`
}

func GenerateImage(prompt string, params structs.ImageParams) string {
	client, err := client.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	model := params.ApiModel
	if model == "" {
		model = "Flux.1-Schnell fp8 (Compact)"
	}

	apiKey := params.ApiKey
	if apiKey == "" {
		apiKey = os.Getenv("AIHORDE_API_KEY")
	}
	if apiKey == "" {
		apiKey = os.Getenv("AI_API_KEY")
	}
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "AI-Horde image generation requires an API key. Get one free at https://stablehorde.net and set AIHORDE_API_KEY env var or use --key")
		os.Exit(1)
	}

	width := params.Width
	if width <= 0 {
		width = 512
	}
	height := params.Height
	if height <= 0 {
		height = 512
	}

	genReq := GenerateRequest{
		Prompt: prompt,
		Models: []string{model},
		Params: &struct {
			Width  int `json:"width,omitempty"`
			Height int `json:"height,omitempty"`
			Steps  int `json:"steps,omitempty"`
		}{
			Width:  width,
			Height: height,
			Steps:  20,
		},
	}

	jsonReq, err := json.Marshal(genReq)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to build request")
		os.Exit(1)
	}

	// Submit generation request
	req, err := http.NewRequest("POST", "https://stablehorde.net/api/v2/generate/async", nil)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred.\nError:", err)
		os.Exit(1)
	}
	req.Body = io.NopCloser(bytes.NewReader(jsonReq))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("apikey", apiKey)
	req.Header.Set("Client-Agent", "tgpt:v2:")

	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred.\nError:", err)
		os.Exit(1)
	}

	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read response")
		os.Exit(1)
	}

	if res.StatusCode != 202 {
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	var genResp GenerateResponse
	if err := json.Unmarshal(body, &genResp); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse response")
		os.Exit(1)
	}

	// Poll for status (max 60 attempts × 2s = ~2 min timeout)
	const maxAttempts = 60
	for attempt := 0; attempt < maxAttempts; attempt++ {
		time.Sleep(2 * time.Second)

		req, err := http.NewRequest("GET", "https://stablehorde.net/api/v2/generate/status/"+genResp.ID, nil)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error checking status:", err)
			os.Exit(1)
		}
		req.Header.Set("apikey", apiKey)
		req.Header.Set("Client-Agent", "tgpt:v2:")

		res, err := client.Do(req)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error checking status:", err)
			os.Exit(1)
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to read status")
			os.Exit(1)
		}

		var status StatusResponse
		if err := json.Unmarshal(body, &status); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to parse status")
			os.Exit(1)
		}

		if status.Faulted {
			fmt.Fprintln(os.Stderr, "Image generation failed")
			os.Exit(1)
		}

		if status.Done && len(status.Generations) > 0 {
			// Download the image
			imgURL := status.Generations[0].Img

			req, err := http.NewRequest("GET", imgURL, nil)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error downloading image:", err)
				os.Exit(1)
			}

			res, err := client.Do(req)
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error downloading image:", err)
				os.Exit(1)
			}
			defer res.Body.Close()

			if res.StatusCode != http.StatusOK {
				body, _ := io.ReadAll(res.Body)
				fmt.Fprintf(os.Stderr, "Error downloading image: HTTP %d: %s\n", res.StatusCode, string(body))
				os.Exit(1)
			}

			filepath := params.Out
			if filepath == "" {
				filepath = utils.RandomString(20) + ".png"
			}

			file, err := os.Create(filepath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error creating file: %v\n", err)
				os.Exit(1)
			}
			defer file.Close()

			_, err = io.Copy(file, res.Body)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error saving image: %v\n", err)
				os.Exit(1)
			}

			return filepath
		}
	}

	fmt.Fprintln(os.Stderr, "Timed out waiting for image generation (~2 min)")
	os.Exit(1)
	return ""
}
