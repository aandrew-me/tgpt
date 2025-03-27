package duckduckgo

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
)

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type RequestData struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

var statusReqMade = false
var vqd = ""
var vqd_hash = ""

func NewRequest(input string, params structs.Params, prevMessages string) (*http.Response, error) {
	client, err := client.NewClient()
	if err != nil {
		return nil, fmt.Errorf("error creating client: %v", err)
	}

	headers := map[string]string{
		"User-Agent":      `Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"Chromium";v="134", "Not:A-Brand";v="24", "Brave";v="134"`,
		"Accept":          "text/event-stream",
		"Accept-Language": "en-US;q=0.7,en;q=0.3",
		"Accept-Encoding": "gzip, deflate, br",
		"Referer":         "https://duckduckgo.com/",
		"Content-Type":    "application/json",
		"Origin":          "https://duckduckgo.com",
		"Connection":      "keep-alive",
		"Cookie":          "dcm=1",
		"Sec-Fetch-Dest":  "empty",
		"Sec-Fetch-Mode":  "cors",
		"Sec-Fetch-Site":  "same-origin",
		"Pragma":          "no-cache",
		"TE":              "trailers",
		"x-vqd-accept":    "1",
		"Cache-Control":   "no-store",
	}

	// We make the status request and get the vqd
	if (!statusReqMade) {
		statusReq, err := http.NewRequest("GET", "https://duckduckgo.com/duckchat/v1/status", nil)
		if err != nil {
			return nil, fmt.Errorf("error creating status request: %v", err)
		}
	
		for key, value := range headers {
			statusReq.Header.Set(key, value)
		}
	
		statusResp, err := client.Do(statusReq)
		if err != nil {
			return nil, fmt.Errorf("error making status request: %v", err)
		}
		defer statusResp.Body.Close()
	
		vqd = statusResp.Header.Get("x-vqd-4")

		res_vqd_hash := statusResp.Header.Get("x-vqd-hash-1")

		res_vqd_hash_decoded_bytes, _ := base64.StdEncoding.DecodeString(res_vqd_hash)

		res_vqd_hash_decoded := string(res_vqd_hash_decoded_bytes)

		server_hashes, _ := extractServerHashes(res_vqd_hash_decoded)

		client_str := map[string]interface{}{
			"client_hashes": []interface{}{`Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/134.0.0.0 Safari/537.36"Chromium";v="134", "Not:A-Brand";v="24", "Brave";v="134"`, "6823"},
		}
	
		client_info, _ := processVQDHash(client_str)

		hash_txt := fmt.Sprintf(`{"server_hashes":%v,%v,"signals":{}}`, server_hashes, client_info)

		hash_data := []byte(hash_txt)

		vqd_hash = base64.StdEncoding.EncodeToString(hash_data)

		statusReqMade = true
	}

	if vqd != "" {
		headers["x-vqd-4"] = vqd
	}

	headers["x-vqd-hash-1"] = vqd_hash

	// We don't make new status requests after the first one
	// We get the vqd from the main requests afterwards

	delete(headers, "x-vqd-accept")

	// Models
	// "meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo"
	// "mistralai/Mixtral-8x7B-Instruct-v0.1"
	// "claude-3-haiku-20240307"

	model := "o3-mini"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	safeInput, _ := json.Marshal(input)

	var data = strings.NewReader(fmt.Sprintf(`{
		"messages": [
			%v
			{
				"content": %v,
				"role": "user"
			}
		],
		"model": "%v"
	}
	`, params.PrevMessages, string(safeInput), model,))

	req, err := http.NewRequest("POST", "https://duckduckgo.com/duckchat/v1/chat", data)
	if err != nil {
		return nil, fmt.Errorf("error creating chat request: %v", err)
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making chat request: %v", err)
	}

	vqd = resp.Header.Get("x-vqd-4")

	return resp, nil
}

func GetMainText(line string) (mainText string) {
	if len(line) > 6 && line[6] == '{' {
		var dat map[string]interface{}
		if err := json.Unmarshal([]byte(line[6:]), &dat); err == nil {
			if message, ok := dat["message"].(string); ok {
				return strings.ReplaceAll(message, "\\n", "\n")
			}
		}
	}
	return ""
}

func HandleResponse(resp *http.Response) error {
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			mainText := GetMainText(line)
			if mainText != "" {
				fmt.Print(mainText)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading response: %v", err)
	}
	return nil
}

func processVQDHash(input map[string]interface{}) (string, error) {
	ch, exists := input["client_hashes"]
	if !exists {
		return "", fmt.Errorf("Expected an object back from VQD Hash: missing 'client_hashes'")
	}
	clientHashes, ok := ch.([]interface{})
	if !ok {
		return "", fmt.Errorf("Expected 'client_hashes' to be an array")
	}

	newHashes := make([]string, len(clientHashes))
	for i, v := range clientHashes {
		s, ok := v.(string)
		if !ok {
			return "", fmt.Errorf("client_hashes element at index %d is not a string", i)
		}

		data := []byte(s)

		hashBytes := sha256.Sum256(data)

		rawStr := string(hashBytes[:])

		encodedHash := base64.StdEncoding.EncodeToString([]byte(rawStr))

		newHashes[i] = encodedHash
	}

	arrayBytes, err := json.Marshal(newHashes)
	if err != nil {
		return "", err
	}
	result := fmt.Sprintf("\"client_hashes\":%s", arrayBytes)
	return result, nil
}

func extractServerHashes(input string) (string, error) {
	re := regexp.MustCompile(`server_hashes:\s*(\[[^\]]+\])`)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 2 {
		return "", fmt.Errorf("server_hashes not found")
	}

	return matches[1], nil
}