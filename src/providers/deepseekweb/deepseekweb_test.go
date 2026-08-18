package deepseekweb

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSolvePoW(t *testing.T) {
	runtime, _ := findJSRuntime()
	if runtime == "" {
		t.Skip("No JavaScript runtime (bun, node, or deno) found in PATH; skipping PoW test")
	}

	salt := "ca4cac59aed9c714506b"
	expireAt := int64(1787047163088)
	difficulty := 144000
	challenge := "48b8ca19d8015666f108b66d01d647269db9ab5d69d60fcd07ddbaa5b9c391e6"

	answer := SolvePoW(salt, expireAt, difficulty, challenge)
	if answer != 32012 {
		t.Fatalf("expected nonce 32012, got %d", answer)
	}
}

func TestGetMainText(t *testing.T) {
	// Standard SSE chunk with p: response/content
	line1 := `data: {"v": "Hello world", "p": "response/content", "o": "APPEND"}`
	if res := GetMainText(line1); res != "Hello world" {
		t.Fatalf("expected 'Hello world', got %q", res)
	}

	// Thinking chunk
	line2 := `data: {"v": "Thinking...", "p": "response/thinking_content", "o": "APPEND"}`
	if res := GetMainText(line2); res != "Thinking..." {
		t.Fatalf("expected 'Thinking...', got %q", res)
	}

	// Choice format
	line3 := `data: {"choices":[{"delta":{"content":"Choice text"}}]}`
	if res := GetMainText(line3); res != "Choice text" {
		t.Fatalf("expected 'Choice text', got %q", res)
	}

	// API Error format
	lineErr := `{"code":40300,"msg":"MISSING_HEADER","data":null}`
	if res := GetMainText(lineErr); !strings.Contains(res, "MISSING_HEADER") {
		t.Fatalf("expected error message containing MISSING_HEADER, got %q", res)
	}

	// Done signal
	line4 := `data: [DONE]`
	if res := GetMainText(line4); res != "" {
		t.Fatalf("expected empty string for [DONE], got %q", res)
	}
}

func TestTokenExtraction(t *testing.T) {
	rawJSON := `{"value":"kKraa6dmEzEozFu9+iJs8YlNOvnKKUoDs+m3zSKhdPBvl4/3cc1OGJ26FX1EAr66","__version":"1.0.0"}`
	token := strings.TrimSpace(rawJSON)
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

	if token != "kKraa6dmEzEozFu9+iJs8YlNOvnKKUoDs+m3zSKhdPBvl4/3cc1OGJ26FX1EAr66" {
		t.Fatalf("expected extracted token, got %q", token)
	}
}


