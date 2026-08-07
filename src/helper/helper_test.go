package helper

import (
	stdhttp "net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/tools"
	http "github.com/bogdanfinn/fhttp"
)

// ============================================================
// Tests for handleStatus400 body leak (rotate-bugs.md #1)
// ============================================================

// trackedBody wraps an io.ReadCloser and calls onClose when Close() is invoked.
type trackedBody struct {
	reader  *strings.Reader
	onClose func()
}

func (b *trackedBody) Read(p []byte) (n int, err error) {
	return b.reader.Read(p)
}

func (b *trackedBody) Close() error {
	if b.onClose != nil {
		b.onClose()
	}
	return nil
}

// TestHandleStatus400BodyClose verifies that handleStatus400 closes resp.Body.
// Because handleStatus400 calls os.Exit(1), this test runs as a subprocess.
// A close-marker file is created when Body.Close() is called.
// The marker file must exist after handleStatus400 exits (exit code 1).
func TestHandleStatus400BodyClose(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		closeFile := os.Getenv("GO_TEST_CLOSE_FILE")
		body := &trackedBody{
			reader: strings.NewReader(`{"error":"test"}`),
			onClose: func() {
				_ = os.WriteFile(closeFile, []byte("closed"), 0644)
			},
		}
		resp := &http.Response{
			Body:       body,
			StatusCode: 400,
		}
		handleStatus400(resp)
		return
	}

	closeFile := filepath.Join(t.TempDir(), "body_closed.txt")
	cmd := exec.Command(os.Args[0],
		"-test.run=^TestHandleStatus400BodyClose$",
	)
	cmd.Env = append(os.Environ(),
		"GO_TEST_SUBPROCESS=1",
		"GO_TEST_CLOSE_FILE="+closeFile,
	)
	// Ignore the error — handleStatus400 calls os.Exit(1)
	_ = cmd.Run()

	if _, err := os.Stat(closeFile); err != nil {
		t.Fatalf("handleStatus400 did not close resp.Body: %v", err)
	}
}

// TestHandleStatus400ExitCode verifies handleStatus400 exits with code 1.
func TestHandleStatus400ExitCode(t *testing.T) {
	if os.Getenv("GO_TEST_SUBPROCESS") == "1" {
		resp := &http.Response{
			Body:       &trackedBody{reader: strings.NewReader("err")},
			StatusCode: 400,
		}
		handleStatus400(resp)
		return
	}

	cmd := exec.Command(os.Args[0],
		"-test.run=^TestHandleStatus400ExitCode$",
	)
	cmd.Env = append(os.Environ(), "GO_TEST_SUBPROCESS=1")

	err := cmd.Run()
	if err == nil {
		t.Fatal("expected subprocess to exit with code 1, but it succeeded")
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if exitErr.ExitCode() != 1 {
			t.Errorf("expected exit code 1, got %d", exitErr.ExitCode())
		}
	} else {
		t.Errorf("unexpected error type: %T", err)
	}
}

func TestInteractiveStatusErrorDoesNotAppendAssistantMessage(t *testing.T) {
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.WriteHeader(stdhttp.StatusTooManyRequests)
		_, _ = w.Write([]byte(`{"error":"rate limited"}`))
	}))
	defer server.Close()

	params := structs.Params{
		Provider: "openai",
		Url:      server.URL,
	}
	extraOptions := structs.ExtraOptions{
		IsInteractive: true,
		IsGetSilent:   true,
	}

	if response, _, err := MakeRequestAndGetData("hello", params, extraOptions); err == nil {
		t.Fatal("expected interactive 4xx response to return an error")
	} else if response != "" {
		t.Fatalf("expected empty response text on error, got %q", response)
	}

	messages, response := GetData("hello", params, extraOptions)
	if response != "" {
		t.Fatalf("expected empty response text from GetData on error, got %q", response)
	}
	if len(messages) != 0 {
		t.Fatalf("expected no conversation history entries on interactive 4xx, got %#v", messages)
	}
}

func TestGetToolsSystemPrompt(t *testing.T) {
	prompt := GetToolsSystemPrompt()
	if !strings.Contains(prompt, "You are tgpt, a terminal assistant. Today is") {
		t.Fatalf("expected prompt to contain assistant identity and date, got %q", prompt)
	}
	if !strings.Contains(prompt, "The shell environment you are in is") {
		t.Fatalf("expected prompt to contain shell info, got %q", prompt)
	}
	if !strings.Contains(prompt, "The operating system you are on is") {
		t.Fatalf("expected prompt to contain OS info, got %q", prompt)
	}
}

func TestToolDepthLimitClearsToolsAndReturnsResponse(t *testing.T) {
	var receivedToolsField string
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		buf := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(buf)
		bodyStr := string(buf)

		if strings.Contains(bodyStr, `"tools":`) {
			receivedToolsField = "present"
		} else {
			receivedToolsField = "absent"
		}

		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"Final summary response\"}}]}\n\n"))
	}))
	defer server.Close()

	params := structs.Params{
		Provider: "openai",
		Url:      server.URL,
		Tools:    []any{"dummy_tool"},
	}
	extraOptions := structs.ExtraOptions{
		IsNormal:  true,
		ToolDepth: 5,
	}

	res, _, err := MakeRequestAndGetData("hello", params, extraOptions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedToolsField != "absent" {
		t.Errorf("expected tools field to be absent in request payload when ToolDepth >= 5, got %s", receivedToolsField)
	}

	if res != "Final summary response" {
		t.Errorf("expected response 'Final summary response', got %q", res)
	}
}

func TestToolExecutionAutoExec(t *testing.T) {
	step := 0
	server := httptest.NewServer(stdhttp.HandlerFunc(func(w stdhttp.ResponseWriter, r *stdhttp.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		if step == 0 {
			step++
			resp := `{"choices":[{"delta":{"tool_calls":[{"index":0,"id":"call_1","type":"function","function":{"name":"execute_command","arguments":"{\"command\":\"echo auto_exec_test\"}"}}]}}]}`
			_, _ = w.Write([]byte("data: " + resp + "\n\n"))
		} else {
			resp := `{"choices":[{"delta":{"content":"Command executed successfully"}}]} `
			_, _ = w.Write([]byte("data: " + resp + "\n\n"))
		}
	}))
	defer server.Close()

	tools.DefaultRegistry.RegisterBuiltinTools("execute_command")

	params := structs.Params{
		Provider: "openai",
		Url:      server.URL,
		Tools:    tools.DefaultRegistry.GetOpenAITools(),
	}
	extraOptions := structs.ExtraOptions{
		IsNormal: true,
		AutoExec: true,
	}

	res, turnMsgs, err := MakeRequestAndGetData("run command", params, extraOptions)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(res, "Command executed successfully") {
		t.Errorf("expected response to contain 'Command executed successfully', got %q", res)
	}

	foundToolResult := false
	for _, msg := range turnMsgs {
		if tm, ok := msg.(structs.ToolMessage); ok {
			if strings.Contains(tm.Content, "auto_exec_test") {
				foundToolResult = true
				break
			}
		}
	}
	if !foundToolResult {
		t.Errorf("expected turn messages to contain tool result with auto_exec_test, got %#v", turnMsgs)
	}
}

func TestFormatToolArgsTruncation(t *testing.T) {
	longVal := strings.Repeat("a", 150)
	rawArgs := `{"short":"hello","long":"` + longVal + `"}`
	formatted := formatToolArgs(rawArgs)

	if len(formatted) > 130 {
		t.Fatalf("expected formatted string length <= 130, got %d: %q", len(formatted), formatted)
	}
	if !strings.Contains(formatted, "...") {
		t.Fatalf("expected formatted string to contain '...', got %q", formatted)
	}
}

func TestDetectPackageManager(t *testing.T) {
	tests := []struct {
		name          string
		execPath      string
		goos          string
		wantIsPkg     bool
		wantPkgName   string
		wantUpdateCmd string
	}{
		{
			name:          "Scoop on Windows",
			execPath:      `C:\Users\user\scoop\shims\tgpt.exe`,
			wantIsPkg:     true,
			wantPkgName:   "Scoop",
			wantUpdateCmd: "scoop update tgpt",
		},
		{
			name:          "Chocolatey on Windows",
			execPath:      `C:\ProgramData\chocolatey\bin\tgpt.exe`,
			wantIsPkg:     true,
			wantPkgName:   "Chocolatey",
			wantUpdateCmd: "choco upgrade tgpt",
		},
		{
			name:          "Homebrew Cellar",
			execPath:      `/opt/homebrew/Cellar/tgpt/2.13.0/bin/tgpt`,
			wantIsPkg:     true,
			wantPkgName:   "Homebrew",
			wantUpdateCmd: "brew upgrade tgpt",
		},
		{
			name:          "Go bin",
			execPath:      `/home/user/go/bin/tgpt`,
			wantIsPkg:     true,
			wantPkgName:   "Go",
			wantUpdateCmd: "go install github.com/aandrew-me/tgpt/v2@latest",
		},
		{
			name:          "PowerShell install script path on Windows",
			execPath:      `C:\Users\user\AppData\Local\tgpt\tgpt.exe`,
			wantIsPkg:     false,
			wantPkgName:   "",
			wantUpdateCmd: "",
		},
		{
			name:          "Bash install script path on Linux",
			execPath:      `/usr/local/bin/tgpt`,
			wantIsPkg:     false,
			wantPkgName:   "",
			wantUpdateCmd: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIsPkg, gotPkgName, gotUpdateCmd := DetectPackageManager(filepath.FromSlash(tt.execPath))
			if gotIsPkg != tt.wantIsPkg {
				t.Errorf("DetectPackageManager(%q) isPkg = %v, want %v", tt.execPath, gotIsPkg, tt.wantIsPkg)
			}
			if gotPkgName != tt.wantPkgName {
				t.Errorf("DetectPackageManager(%q) pkgName = %q, want %q", tt.execPath, gotPkgName, tt.wantPkgName)
			}
			if gotUpdateCmd != tt.wantUpdateCmd {
				t.Errorf("DetectPackageManager(%q) updateCmd = %q, want %q", tt.execPath, gotUpdateCmd, tt.wantUpdateCmd)
			}
		})
	}
}

func TestDetectPacmanLinux(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux-specific test")
	}
	// For a non-existent file in /usr/bin/tgpt_fake_test, pacman/dpkg/rpm won't own it
	gotIsPkg, _, _ := DetectPackageManager("/usr/bin/tgpt_fake_test")
	if gotIsPkg {
		t.Errorf("Expected false for non-owned file in /usr/bin, got %v", gotIsPkg)
	}
}

func TestDetectPackageManagerMultiGOPATH(t *testing.T) {
	sep := string(os.PathListSeparator)
	origGOPATH := os.Getenv("GOPATH")
	defer os.Setenv("GOPATH", origGOPATH)

	dummyFirst := filepath.Join("custom", "first_path")
	dummySecond := filepath.Join("custom", "second_path")
	os.Setenv("GOPATH", dummyFirst+sep+dummySecond)

	execInSecond := filepath.Join(dummySecond, "bin", "tgpt")
	isPkg, pkgName, _ := DetectPackageManager(execInSecond)
	if !isPkg || pkgName != "Go" {
		t.Errorf("expected Go detection for path in multi-entry GOPATH (%q), got %v, %q", execInSecond, isPkg, pkgName)
	}
}

func TestEscapePowerShellArg(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{
			input: `C:\Users\name\AppData\Local\tgpt\tgpt.exe`,
			want:  `'C:\Users\name\AppData\Local\tgpt\tgpt.exe'`,
		},
		{
			input: `C:\Users\John's PC\tgpt.exe`,
			want:  `'C:\Users\John''s PC\tgpt.exe'`,
		},
	}

	for _, tt := range tests {
		got := escapePowerShellArg(tt.input)
		if got != tt.want {
			t.Errorf("escapePowerShellArg(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
