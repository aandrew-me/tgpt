package opencode

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	tls "github.com/bogdanfinn/utls"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

var defaultTools = []any{
	map[string]any{"type": "function", "function": map[string]any{"name": "edit"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "experimental_image_generate"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "glob"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "grep"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "question"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "read"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "shell"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "skill"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "subagent"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "webfetch"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "websearch"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "write"}},
	map[string]any{"type": "function", "function": map[string]any{"name": "execute"}},
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage"`
}

type RequestBody struct {
	Model         string         `json:"model"`
	Stream        bool           `json:"stream"`
	Messages      []any          `json:"messages"`
	Temperature   *float64       `json:"temperature,omitempty"`
	TopP          *float64       `json:"top_p,omitempty"`
	Tools         []any          `json:"tools,omitempty"`
	ToolChoice    string         `json:"tool_choice,omitempty"`
	StreamOptions *StreamOptions `json:"stream_options,omitempty"`
}

// bunProfile matches Bun/BoringSSL HTTP/1.1 TLS fingerprint; browser/Go/Node TLS fingerprints get 429 from OpenCode.
func bunProfile() profiles.ClientProfile {
	helloID := tls.ClientHelloID{
		Client:  "Bun",
		Version: "1",
		SpecFactory: func() (tls.ClientHelloSpec, error) {
			return tls.ClientHelloSpec{
				CipherSuites: []uint16{
					tls.TLS_AES_128_GCM_SHA256,
					tls.TLS_AES_256_GCM_SHA384,
					tls.TLS_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
					tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
					tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
					tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
					tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
					tls.TLS_RSA_WITH_AES_128_CBC_SHA,
					tls.TLS_RSA_WITH_AES_256_CBC_SHA,
				},
				CompressionMethods: []byte{0},
				Extensions: []tls.TLSExtension{
					&tls.SNIExtension{},
					&tls.ExtendedMasterSecretExtension{},
					&tls.RenegotiationInfoExtension{Renegotiation: tls.RenegotiateOnceAsClient},
					&tls.SupportedCurvesExtension{Curves: []tls.CurveID{tls.X25519, tls.CurveP256, tls.CurveP384}},
					&tls.SupportedPointsExtension{SupportedPoints: []byte{0}},
					&tls.SessionTicketExtension{},
					&tls.ALPNExtension{AlpnProtocols: []string{"http/1.1"}},
					&tls.StatusRequestExtension{},
					&tls.SignatureAlgorithmsExtension{SupportedSignatureAlgorithms: []tls.SignatureScheme{
						tls.ECDSAWithP256AndSHA256,
						tls.PSSWithSHA256,
						tls.PKCS1WithSHA256,
						tls.ECDSAWithP384AndSHA384,
						tls.PSSWithSHA384,
						tls.PKCS1WithSHA384,
						tls.PSSWithSHA512,
						tls.PKCS1WithSHA512,
						tls.PKCS1WithSHA1,
					}},
					&tls.SCTExtension{},
					&tls.KeyShareExtension{KeyShares: []tls.KeyShare{{Group: tls.X25519}}},
					&tls.PSKKeyExchangeModesExtension{Modes: []uint8{tls.PskModeDHE}},
					&tls.SupportedVersionsExtension{Versions: []uint16{tls.VersionTLS13, tls.VersionTLS12}},
					&tls.UtlsPaddingExtension{GetPaddingLen: tls.BoringPaddingStyle},
				},
			}, nil
		},
	}
	return profiles.NewClientProfile(helloID, nil, nil, nil, 0, nil, nil, 0, true, nil, nil, 0, nil, false)
}

func defaultSystemPrompt() string {
	// OpenCode's free-tier gateway requires the harness prompt + env block; use neutral values to avoid leaking local paths/OS.
	return fmt.Sprintf(`You are an AI agent running in OpenCode, a coding agent harness. Help the user accomplish their goals using the tools you have available.

# Harness
- Responses are rendered as GitHub-flavored Markdown.
- `+"`<system-reminder>`"+` blocks are harness instructions, not user-authored content. Read and follow them.
- Prefer parallelizing independent tool calls.
- Prefer dedicated tools over shell commands; fall back to the shell when a tool cannot do what you need.
- Do not chain shell commands with separators like `+"`echo \"====\";`"+` or `+"`printf '---'`"+`; the output becomes noisy in a way that makes the user's side of the conversation worse.
- Use the write tool to create files or completely replace their content. Prefer using the edit tool for targeted changes.
- Use the edit tool for targeted changes to existing text files. It replaces the exact text in `+"`oldString`"+` with `+"`newString`"+`, and the values must differ. By default, `+"`oldString`"+` must occur exactly once. If it occurs multiple times, include more surrounding context to make it unique or set `+"`replaceAll`"+` to true to replace every occurrence.

# Communication
- Use clear file paths when referring to files.
- Keep responses clear and concise, and avoid unnecessary technical jargon.

# Working in codebases
- Keep changes consistent with the structure, naming, style, and patterns of the surrounding code.
- Treat unfamiliar files or changes as potential user work and investigate before deleting or overwriting them.

Today's date: %s

Here is some useful information about the environment you are running in:
<env>
  Working directory: .
  Workspace root folder: .
  Is directory a git repo: no
  Platform: linux
</env>`, time.Now().Format("Mon Jan 02 2006"))
}

// generateSessionID encodes a descending millisecond timestamp + random base62 suffix matching OpenCode Identifier.
func generateSessionID() string {
	ts := (^(uint64(time.Now().UnixMilli())*0x1000 + 1)) & 0xffffffffffff
	var suffix [14]byte
	max := big.NewInt(int64(len(base62Chars)))
	for i := range suffix {
		n, err := rand.Int(rand.Reader, max)
		if err != nil {
			suffix[i] = '0'
			continue
		}
		suffix[i] = base62Chars[n.Int64()]
	}
	return fmt.Sprintf("ses_%012x%s", ts, string(suffix[:]))
}

var (
	sessionIDPattern = regexp.MustCompile(`^ses_[0-9a-f]{12}[0-9A-Za-z]{14}$`)
	threadSessionsMu sync.Mutex
	threadSessions   = map[string]string{}
)

// resolveSessionID ensures OpenCode receives a valid ses_ ID while keeping interactive ThreadID affinity.
func resolveSessionID(threadID string) string {
	if threadID == "" {
		return generateSessionID()
	}
	if sessionIDPattern.MatchString(threadID) {
		return threadID
	}
	threadSessionsMu.Lock()
	defer threadSessionsMu.Unlock()
	if id, ok := threadSessions[threadID]; ok {
		return id
	}
	id := generateSessionID()
	threadSessions[threadID] = id
	return id
}

func buildRequestBody(model, input string, params structs.Params) RequestBody {
	sysPrompt := defaultSystemPrompt()
	if len(params.Tools) == 0 {
		sysPrompt += "\n\nNo tools are executable in this environment. Answer directly in plain text without calling tools."
	}
	if params.SystemPrompt != "" {
		sysPrompt += "\n\n" + params.SystemPrompt
	}

	messages := make([]any, 0, len(params.PrevMessages)+2)
	messages = append(messages, structs.DefaultMessage{
		Content: sysPrompt,
		Role:    "system",
	})

	tools := make([]any, 0, len(params.Tools)+len(defaultTools))
	seenTools := make(map[string]bool)

	for _, t := range params.Tools {
		if name := toolName(t); name != "" {
			seenTools[name] = true
		}
		tools = append(tools, t)
	}

	// Gateway returns 429 if tools is empty; tool_choice: "none" stops calls when params.Tools is empty.
	for _, dt := range defaultTools {
		if name := toolName(dt); name != "" && !seenTools[name] {
			seenTools[name] = true
			tools = append(tools, dt)
		}
	}

	toolChoice := "none"
	if len(params.Tools) > 0 {
		toolChoice = "auto"
	}

	requestInfo := RequestBody{
		Model:         model,
		Stream:        true,
		Messages:      messages,
		Tools:         tools,
		ToolChoice:    toolChoice,
		StreamOptions: &StreamOptions{IncludeUsage: true},
	}
	requestInfo.Temperature, requestInfo.TopP = params.SamplingOptions()

	if len(params.PrevMessages) > 0 {
		requestInfo.Messages = append(requestInfo.Messages, params.PrevMessages...)
	}

	if input != "" {
		requestInfo.Messages = append(requestInfo.Messages, structs.DefaultMessage{
			Role:    "user",
			Content: input,
		})
	}

	return requestInfo
}

func NewRequest(input string, params structs.Params) (*http.Response, error) {
	httpClient, err := client.NewClientWithOptions(
		0,
		tls_client.WithClientProfile(bunProfile()),
		tls_client.WithForceHttp1(),
	)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	model := "deepseek-v4.1-flash-free"
	if params.ApiModel != "" {
		model = params.ApiModel
	} else if envModel := os.Getenv("OPENCODE_MODEL"); envModel != "" {
		model = envModel
	}

	apiKey := "public"
	if params.ApiKey != "" {
		apiKey = params.ApiKey
	} else if envKey := os.Getenv("OPENCODE_API_KEY"); envKey != "" {
		apiKey = envKey
	} else if envKey := os.Getenv("AI_API_KEY"); envKey != "" {
		apiKey = envKey
	}

	url := params.Url
	if url == "" {
		if envUrl := os.Getenv("OPENCODE_URL"); envUrl != "" {
			url = envUrl + "/v1/chat/completions"
		}
	}

	if url == "" {
		url = "https://opencode.ai/inference/openai/v1/chat/completions"
	}

	requestInfo := buildRequestBody(model, input, params)
	jsonRequest, err := json.Marshal(requestInfo)

	if err != nil {
		log.Fatal("Failed to build user request")
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonRequest))
	if err != nil {
		log.Fatal("Some error has occured.\nError:", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "*/*")

	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}

	projectID, _ := utils.GenerateHexString(40)
	sessionID := resolveSessionID(params.ThreadID)

	req.Header.Set("x-opencode-client", "cli")
	req.Header.Set("x-opencode-project", projectID)
	req.Header.Set("User-Agent", "opencode/latest/2.2.7/cli")
	req.Header.Set("x-opencode-session", sessionID)
	req.Header.Set("x-opencode-session-id", sessionID)
	req.Header.Set("x-session-affinity", sessionID)
	req.Header.Set("X-Session-Id", sessionID)

	return httpClient.Do(req)
}

func GetMainText(line string) (mainText string) {
	var obj = "{}"
	if after, ok := strings.CutPrefix(line, "data: "); ok {
		obj = after
	}

	var d structs.CommonResponse
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if len(d.Choices) > 0 {
		mainText = d.Choices[0].Delta.Content
		return mainText
	}
	return ""
}

func toolName(t any) string {
	switch v := t.(type) {
	case map[string]any:
		if fn, ok := v["function"].(map[string]any); ok {
			if name, ok := fn["name"].(string); ok {
				return name
			}
		}
	}
	b, err := json.Marshal(t)
	if err == nil {
		var s struct {
			Function struct {
				Name string `json:"name"`
			} `json:"function"`
		}
		if json.Unmarshal(b, &s) == nil {
			return s.Function.Name
		}
	}
	return ""
}
