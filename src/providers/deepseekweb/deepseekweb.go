package deepseekweb

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
)

//go:embed sha3_wasm_bg.wasm
var wasmBytes []byte

// findJSRuntime locates an available JavaScript runtime (bun, node, or deno).
func findJSRuntime() (string, []string) {
	if custom := os.Getenv("DEEPSEEK_WEB_RUNTIME"); custom != "" {
		parts := strings.Fields(custom)
		if len(parts) > 0 {
			return parts[0], parts[1:]
		}
	}
	for _, r := range []string{"bun", "node", "deno"} {
		if path, err := exec.LookPath(r); err == nil && path != "" {
			if r == "deno" {
				return "deno", []string{"eval", "--allow-env"}
			}
			return r, []string{"-e"}
		}
	}
	return "", nil
}

// SolvePoW solves the DeepSeekHashV1 challenge using the embedded WebAssembly solver via Node/Bun/Deno.
func SolvePoW(salt string, expireAt int64, difficulty int, challenge string) int {
	prefix := fmt.Sprintf("%s_%d_", salt, expireAt)
	if difficulty <= 0 {
		difficulty = 150000
	}

	runtime, args := findJSRuntime()
	if runtime == "" {
		fmt.Fprintln(os.Stderr, "Warning: No JavaScript runtime (node, bun, or deno) found in PATH to solve DeepSeek Web Proof-of-Work.")
		return -1
	}

	if len(wasmBytes) > 0 {
		script := `
const chunks=[];
process.stdin.on('data',c=>chunks.push(c));
process.stdin.on('end',()=>{
	const total=chunks.reduce((acc,c)=>acc+c.length,0);
	const wasm=new Uint8Array(total);
	let offset=0;
	for(const c of chunks){wasm.set(c,offset);offset+=c.length;}
	WebAssembly.instantiate(wasm,{__wbindgen_placeholder__:{}}).then(m=>{
		const exp=m.instance.exports;
		const enc=new TextEncoder();
		function pass(s){
			const b=enc.encode(s);
			const p=exp.__wbindgen_export_0(b.length,1);
			new Uint8Array(exp.memory.buffer).set(b,p);
			return [p,b.length];
		}
		const [cp,cl]=pass(process.env.DS_CHALLENGE||"");
		const [pp,pl]=pass(process.env.DS_PREFIX||"");
		const rp=exp.__wbindgen_add_to_stack_pointer(-16);
		exp.wasm_solve(rp,cp,cl,pp,pl,parseFloat(process.env.DS_DIFFICULTY||"0"));
		const ans=(new Float64Array(exp.memory.buffer,rp+8,1))[0];
		console.log(Math.round(ans));
		process.exit(0);
	}).catch(()=>{console.log(-1);process.exit(0);});
});`

		if runtime == "deno" {
			script = "import process from 'node:process';\n" + script
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		cmdArgs := append(args, script)
		cmd := exec.CommandContext(ctx, runtime, cmdArgs...)
		cmd.Stdin = bytes.NewReader(wasmBytes)
		cmd.Env = append(os.Environ(),
			"DS_CHALLENGE="+challenge,
			"DS_PREFIX="+prefix,
			"DS_DIFFICULTY="+strconv.Itoa(difficulty),
		)
		if out, err := cmd.Output(); err == nil {
			trimmed := strings.TrimSpace(string(out))
			if ans, err := strconv.Atoi(trimmed); err == nil && ans >= 0 {
				return ans
			}
		}
	}

	return -1
}

type powChallengeResp struct {
	Code int `json:"code"`
	Data struct {
		BizData struct {
			Challenge struct {
				Algorithm  string `json:"algorithm"`
				Challenge  string `json:"challenge"`
				Salt       string `json:"salt"`
				ExpireAt   int64  `json:"expire_at"`
				Difficulty int    `json:"difficulty"`
				Signature  string `json:"signature"`
				TargetPath string `json:"target_path"`
			} `json:"challenge"`
		} `json:"biz_data"`
	} `json:"data"`
}

type powResponsePayload struct {
	Algorithm  string `json:"algorithm"`
	Challenge  string `json:"challenge"`
	Salt       string `json:"salt"`
	Answer     int    `json:"answer"`
	Signature  string `json:"signature"`
	TargetPath string `json:"target_path"`
}

type createSessionResp struct {
	Code int `json:"code"`
	Data struct {
		BizData struct {
			ID string `json:"id"`
		} `json:"biz_data"`
	} `json:"data"`
}

type completionRequest struct {
	ChatSessionID   string `json:"chat_session_id"`
	ParentMessageID any    `json:"parent_message_id"`
	Prompt          string `json:"prompt"`
	RefFileIDs      []any  `json:"ref_file_ids"`
	ThinkingEnabled bool   `json:"thinking_enabled"`
	SearchEnabled   bool   `json:"search_enabled"`
}

// RequestPoWHeader requests a PoW challenge from DeepSeek and solves it.
func RequestPoWHeader(client tls_client.HttpClient, baseUrl string, token string) string {
	challengeUrl := baseUrl + "/api/v0/chat/create_pow_challenge"
	bodyBytes, _ := json.Marshal(map[string]string{
		"target_path": "/api/v0/chat/completion",
	})

	req, err := http.NewRequest("POST", challengeUrl, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Origin", baseUrl)
	req.Header.Set("Referer", baseUrl+"/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("x-app-version", "20241129.0")

	resp, err := client.Do(req)
	if err != nil || resp == nil {
		return ""
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var parsed powChallengeResp
	if err := json.Unmarshal(respData, &parsed); err != nil {
		return ""
	}

	ch := parsed.Data.BizData.Challenge
	if ch.Algorithm != "DeepSeekHashV1" || ch.Challenge == "" {
		return ""
	}

	answer := SolvePoW(ch.Salt, ch.ExpireAt, ch.Difficulty, ch.Challenge)
	if answer == -1 {
		return ""
	}

	powResp := powResponsePayload{
		Algorithm:  ch.Algorithm,
		Challenge:  ch.Challenge,
		Salt:       ch.Salt,
		Answer:     answer,
		Signature:  ch.Signature,
		TargetPath: ch.TargetPath,
	}

	jsonPow, err := json.Marshal(powResp)
	if err != nil {
		return ""
	}

	return base64.StdEncoding.EncodeToString(jsonPow)
}

// CreateSession initiates a new chat session on DeepSeek web.
func CreateSession(client tls_client.HttpClient, baseUrl string, token string) string {
	sessionUrl := baseUrl + "/api/v0/chat_session/create"
	bodyBytes, _ := json.Marshal(map[string]string{
		"agent": "chat",
	})

	req, err := http.NewRequest("POST", sessionUrl, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return ""
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Origin", baseUrl)
	req.Header.Set("Referer", baseUrl+"/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("x-app-version", "20241129.0")

	resp, err := client.Do(req)
	if err != nil || resp == nil {
		return ""
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}

	var parsed createSessionResp
	if err := json.Unmarshal(respData, &parsed); err != nil {
		return ""
	}

	return parsed.Data.BizData.ID
}

var (
	currentSessionID    string
	lastParentMessageID any
)

// NewRequest sends a prompt to DeepSeek web API and returns the streaming response.
func NewRequest(input string, params structs.Params) (*http.Response, error) {
	httpClient, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	token := params.ApiKey
	if token == "" {
		token = os.Getenv("DEEPSEEK_WEB_TOKEN")
	}

	token = strings.TrimSpace(token)
	token = strings.Trim(token, "\"'`")
	token = strings.TrimPrefix(token, "Bearer ")
	if strings.HasPrefix(token, "{") {
		var tokenObj struct {
			Value   string `json:"value"`
			Token   string `json:"token"`
			Version any    `json:"__version"`
		}
		if err := json.Unmarshal([]byte(token), &tokenObj); err == nil {
			if tokenObj.Value != "" {
				token = tokenObj.Value
			} else if tokenObj.Token != "" {
				token = tokenObj.Token
			}
		}
		token = strings.TrimSpace(token)
		token = strings.Trim(token, "\"'`")
		token = strings.TrimPrefix(token, "Bearer ")
	}

	if token == "" {
		fmt.Fprintln(os.Stderr, "Error: deepseek-web requires a user token from chat.deepseek.com.")
		fmt.Fprintln(os.Stderr, "Set DEEPSEEK_WEB_TOKEN environment variable or pass --key <userToken>.")
		os.Exit(1)
	}

	baseUrl := params.Url
	if baseUrl == "" {
		baseUrl = os.Getenv("DEEPSEEK_WEB_URL")
	}
	if baseUrl == "" {
		baseUrl = os.Getenv("DEEPSEEK_WEB_BASE_URL")
	}
	if baseUrl == "" {
		baseUrl = "https://chat.deepseek.com"
	}
	baseUrl = strings.TrimSuffix(baseUrl, "/")

	thinkingEnv := strings.ToLower(os.Getenv("DEEPSEEK_WEB_THINKING"))
	thinkingEnabled := thinkingEnv == "true" || thinkingEnv == "1"

	searchEnv := strings.ToLower(os.Getenv("DEEPSEEK_WEB_SEARCH"))
	searchEnabled := searchEnv == "true" || searchEnv == "1"

	// If starting a new session or no active session exists, create one
	if len(params.PrevMessages) == 0 || currentSessionID == "" {
		currentSessionID = CreateSession(httpClient, baseUrl, token)
		if currentSessionID == "" {
			return nil, fmt.Errorf("failed to create chat session on DeepSeek web (verify DEEPSEEK_WEB_TOKEN or network connection)")
		}
		lastParentMessageID = nil
	}

	// Construct final prompt without repeating previous conversation history
	var finalPrompt string
	if params.SystemPrompt != "" && lastParentMessageID == nil {
		finalPrompt = "System: " + params.SystemPrompt + "\n\n" + input
	} else {
		finalPrompt = input
	}

	powHeader := RequestPoWHeader(httpClient, baseUrl, token)
	if powHeader == "" {
		fmt.Fprintln(os.Stderr, "Warning: Failed to generate Proof-of-Work header for DeepSeek web.")
	}

	completionBody := completionRequest{
		ChatSessionID:   currentSessionID,
		ParentMessageID: lastParentMessageID,
		Prompt:          finalPrompt,
		RefFileIDs:      []any{},
		ThinkingEnabled: thinkingEnabled,
		SearchEnabled:   searchEnabled,
	}

	jsonRequest, err := json.Marshal(completionBody)
	if err != nil {
		log.Fatal("Failed to build user request")
	}

	completionUrl := baseUrl + "/api/v0/chat/completion"
	req, err := http.NewRequest("POST", completionUrl, bytes.NewBuffer(jsonRequest))
	if err != nil {
		log.Fatal("Some error has occurred.\nError:", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Origin", baseUrl)
	req.Header.Set("Referer", baseUrl+"/")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("x-app-version", "20241129.0")
	req.Header.Set("Authorization", "Bearer "+token)

	if powHeader != "" {
		req.Header.Set("x-ds-pow-response", powHeader)
	}

	return httpClient.Do(req)
}

type deepSeekWebStreamChunk struct {
	Code    int    `json:"code,omitempty"`
	Msg     string `json:"msg,omitempty"`
	Message string `json:"message,omitempty"`
	Error   string `json:"error,omitempty"`
	V       any    `json:"v,omitempty"`
	P       string `json:"p,omitempty"`
	O       string `json:"o,omitempty"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

// GetMainText parses SSE events from the DeepSeek web streaming response.
func GetMainText(line string) string {
	if strings.Contains(line, "[DONE]") {
		return ""
	}

	obj := line
	if strings.HasPrefix(line, "data: ") {
		obj = strings.TrimPrefix(line, "data: ")
	} else if strings.Contains(line, "data:") {
		parts := strings.SplitN(line, "data:", 2)
		if len(parts) > 1 {
			obj = strings.TrimSpace(parts[1])
		}
	}

	// Update lastParentMessageID if response message ID is present in stream metadata
	var meta struct {
		ResponseMessageID any `json:"response_message_id"`
		V                 struct {
			Response struct {
				MessageID any `json:"message_id"`
			} `json:"response"`
		} `json:"v"`
	}
	if err := json.Unmarshal([]byte(obj), &meta); err == nil {
		if meta.ResponseMessageID != nil {
			lastParentMessageID = meta.ResponseMessageID
		} else if meta.V.Response.MessageID != nil {
			lastParentMessageID = meta.V.Response.MessageID
		}
	}

	var d deepSeekWebStreamChunk
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if d.Code != 0 && d.Msg != "" {
		return fmt.Sprintf("\n[DeepSeek Web Error %d: %s]\n", d.Code, d.Msg)
	}
	if d.Error != "" {
		return fmt.Sprintf("\n[DeepSeek Web Error: %s]\n", d.Error)
	}
	if d.Message != "" {
		return fmt.Sprintf("\n[DeepSeek Web Error: %s]\n", d.Message)
	}

	if len(d.Choices) > 0 {
		return d.Choices[0].Delta.Content
	}

	if str, ok := d.V.(string); ok && str != "" {
		if d.P == "response/content" || d.P == "response/thinking_content" || d.P == "" {
			return str
		}
	}

	return ""
}


