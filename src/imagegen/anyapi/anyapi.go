package anyapi

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
)

type ImageRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	N      int    `json:"n,omitempty"`
}

type ImageResponse struct {
	Created int `json:"created"`
	Data    []struct {
		B64JSON       string `json:"b64_json"`
		RevisedPrompt string `json:"revised_prompt,omitempty"`
	} `json:"data"`
}

func GenerateImage(prompt string, params structs.ImageParams) string {
	client, err := client.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	model := params.ApiModel
	if model == "" {
		model = "google/gemini-2.5-flash-image"
	}

	apiKey := params.ApiKey
	if apiKey == "" {
		apiKey = os.Getenv("ANYAPI_API_KEY")
	}

	requestInfo := ImageRequest{
		Model:  model,
		Prompt: prompt,
		N:      1,
	}

	jsonRequest, err := json.Marshal(requestInfo)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to build request")
		os.Exit(1)
	}

	req, err := http.NewRequest("POST", "https://api.anyapi.ai/v1/images/generations", bytes.NewBuffer(jsonRequest))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred.\nError:", err)
		os.Exit(1)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	res, err := client.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred.\nError:", err)
		os.Exit(1)
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		body, _ := io.ReadAll(res.Body)
		fmt.Fprintf(os.Stderr, "Error: %s\n", string(body))
		os.Exit(1)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to read response")
		os.Exit(1)
	}

	var result ImageResponse
	if err := json.Unmarshal(body, &result); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to parse response")
		os.Exit(1)
	}

	if len(result.Data) == 0 {
		fmt.Fprintln(os.Stderr, "No image data in response")
		os.Exit(1)
	}

	filepath := params.Out
	if filepath == "" {
		randId := utils.RandomString(20)
		filepath = randId + ".png"
	}

	decoded, err := base64.StdEncoding.DecodeString(result.Data[0].B64JSON)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Failed to decode image data")
		os.Exit(1)
	}

	if err := os.WriteFile(filepath, decoded, 0644); err != nil {
		fmt.Fprintln(os.Stderr, "Failed to save image")
		os.Exit(1)
	}

	return filepath
}
