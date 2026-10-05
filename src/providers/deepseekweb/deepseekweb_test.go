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

	// Initial response object with fragments
	lineFragInit := `data: {"v":{"response":{"message_id":2,"parent_id":1,"model":"","role":"ASSISTANT","status":"WIP","fragments":[{"id":2,"type":"RESPONSE","content":"My"}]}}}`
	if res := GetMainText(lineFragInit); res != "My" {
		t.Fatalf("expected 'My', got %q", res)
	}

	// Fragment append patch
	lineFragAppend := `data: {"p":"response/fragments/-1/content","o":"APPEND","v":" name"}`
	if res := GetMainText(lineFragAppend); res != " name" {
		t.Fatalf("expected ' name', got %q", res)
	}

	// Raw v shorthand token
	lineShorthand := `data: {"v":" is"}`
	if res := GetMainText(lineShorthand); res != " is" {
		t.Fatalf("expected ' is', got %q", res)
	}

	// Fragment append patch with array of fragments
	lineFragArrayAppend := `data: {"p":"response/fragments","o":"APPEND","v":[{"type":"RESPONSE","content":"Hi"}]}`
	if res := GetMainText(lineFragArrayAppend); res != "Hi" {
		t.Fatalf("expected 'Hi', got %q", res)
	}

	// Fragment append patch with single object fragment
	lineFragObjectAppend := `data: {"p":"response/fragments","o":"APPEND","v":{"type":"RESPONSE","content":"Hello"}}`
	if res := GetMainText(lineFragObjectAppend); res != "Hello" {
		t.Fatalf("expected 'Hello', got %q", res)
	}

	// Non-content updates (status, tokens) should return empty string
	lineStatus := `data: {"p":"response/status","o":"SET","v":"FINISHED"}`
	if res := GetMainText(lineStatus); res != "" {
		t.Fatalf("expected empty string for status update, got %q", res)
	}

	// Initial response object with fragments directly under v
	lineFragDirectV := `data: {"v":{"fragments":[{"type":"RESPONSE","content":"DirectV"}]}}`
	if res := GetMainText(lineFragDirectV); res != "DirectV" {
		t.Fatalf("expected 'DirectV', got %q", res)
	}

	// Initial response object with fragments directly under response
	lineFragDirectResp := `data: {"response":{"fragments":[{"type":"RESPONSE","content":"DirectResp"}]}}`
	if res := GetMainText(lineFragDirectResp); res != "DirectResp" {
		t.Fatalf("expected 'DirectResp', got %q", res)
	}

	// Fragment append patch targeting indexed fragment path with object
	lineFragIndexedObj := `data: {"p":"response/fragments/0","o":"APPEND","v":{"type":"RESPONSE","content":" indexed obj"}}`
	if res := GetMainText(lineFragIndexedObj); res != " indexed obj" {
		t.Fatalf("expected ' indexed obj', got %q", res)
	}

	// Fragment append patch targeting indexed fragment path with string
	lineFragIndexedStr := `data: {"p":"response/fragments/-1","o":"APPEND","v":" indexed str"}`
	if res := GetMainText(lineFragIndexedStr); res != " indexed str" {
		t.Fatalf("expected ' indexed str', got %q", res)
	}

	// Fragment with thinking_content fallback
	lineFragThinking := `data: {"v":{"fragments":[{"type":"THINKING","thinking_content":"Reasoning"}]}}`
	if res := GetMainText(lineFragThinking); res != "Reasoning" {
		t.Fatalf("expected 'Reasoning', got %q", res)
	}

	// Fragment with text fallback
	lineFragText := `data: {"v":{"fragments":[{"type":"RESPONSE","text":"FallbackText"}]}}`
	if res := GetMainText(lineFragText); res != "FallbackText" {
		t.Fatalf("expected 'FallbackText', got %q", res)
	}

	// REQUEST fragments in conversation history should be filtered out
	lineFragWithRequest := `data: {"v":{"response":{"fragments":[{"type":"REQUEST","content":"User prompt"},{"type":"RESPONSE","content":"Assistant output"}]}}}`
	if res := GetMainText(lineFragWithRequest); res != "Assistant output" {
		t.Fatalf("expected 'Assistant output', got %q", res)
	}

	// String array in fragments patch
	lineFragStringArray := `data: {"p":"response/fragments","o":"APPEND","v":["Chunk1","Chunk2"]}`
	if res := GetMainText(lineFragStringArray); res != "Chunk1Chunk2" {
		t.Fatalf("expected 'Chunk1Chunk2', got %q", res)
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


