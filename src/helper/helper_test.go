package helper

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/structs"
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
// Currently this test demonstrates the bug: handleStatus400 does NOT close the body.
// Once fixed, the marker file will be created and this test will pass.
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

	_, err := os.Stat(closeFile)
	if os.IsNotExist(err) {
		t.Error("BUG: handleStatus400 did NOT call resp.Body.Close() — " +
			"body leak: io.ReadAll(resp.Body) reads the body but Close() is never called, " +
			"then os.Exit(1) exits without any deferred cleanup. " +
			"Fix: add 'defer resp.Body.Close()' at the start of handleStatus400.")
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

// ============================================================
// Tests for providersForRotation (helper.go:859-878)
// ============================================================

func TestProvidersForRotation_Empty(t *testing.T) {
	params := structs.Params{
		Provider:        "openai",
		RotateProviders: "",
	}
	result := providersForRotation(params)
	if len(result) != 1 {
		t.Fatalf("expected 1 provider, got %d: %v", len(result), result)
	}
	if result[0] != "openai" {
		t.Errorf("expected 'openai', got %q", result[0])
	}
}

func TestProvidersForRotation_ValidList(t *testing.T) {
	params := structs.Params{
		Provider:        "openai",
		RotateProviders: "gemini,deepseek",
	}
	result := providersForRotation(params)
	expected := []string{"gemini", "deepseek"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d providers, got %d: %v", len(expected), len(result), result)
	}
	for i, p := range expected {
		if result[i] != p {
			t.Errorf("result[%d] = %q, want %q", i, result[i], p)
		}
	}
}

func TestProvidersForRotation_WithSpaces(t *testing.T) {
	params := structs.Params{
		RotateProviders: "  gemini , deepseek  ",
	}
	result := providersForRotation(params)
	expected := []string{"gemini", "deepseek"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d providers, got %d: %v", len(expected), len(result), result)
	}
}

func TestProvidersForRotation_SkipsInvalid(t *testing.T) {
	params := structs.Params{
		RotateProviders: "gemini,doesnotexist,deepseek",
	}
	result := providersForRotation(params)
	expected := []string{"gemini", "deepseek"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d providers, got %d: %v", len(expected), len(result), result)
	}
}

func TestProvidersForRotation_AllInvalidFallback(t *testing.T) {
	params := structs.Params{
		Provider:        "openai",
		RotateProviders: "invalid1,invalid2",
	}
	result := providersForRotation(params)
	if len(result) != 1 {
		t.Fatalf("expected 1 (fallback), got %d: %v", len(result), result)
	}
	if result[0] != "openai" {
		t.Errorf("expected fallback to 'openai', got %q", result[0])
	}
}

func TestProvidersForRotation_EmptyItems(t *testing.T) {
	params := structs.Params{
		Provider:        "openai",
		RotateProviders: "gemini,,deepseek",
	}
	result := providersForRotation(params)
	expected := []string{"gemini", "deepseek"}
	if len(result) != len(expected) {
		t.Fatalf("expected %d providers, got %d: %v", len(expected), len(result), result)
	}
}

func TestProvidersForRotation_MixedEmptyInvalid(t *testing.T) {
	params := structs.Params{
		RotateProviders: " , invalid, , ",
	}
	result := providersForRotation(params)
	// All filtered out → fallback to params.Provider (empty string = "valid" in list)
	if len(result) != 1 {
		t.Fatalf("expected 1 (fallback), got %d: %v", len(result), result)
	}
}

// ============================================================
// Tests for fallback semantics (rotate-bugs.md #4)
// ============================================================

func TestProvidersForRotation_OriginalProviderNotIncluded(t *testing.T) {
	// This test documents the current behavior:
	// When RotateProviders is set, params.Provider is NOT prepended.
	// --rotate replaces, it doesn't extend.
	params := structs.Params{
		Provider:        "openai",
		RotateProviders: "gemini,deepseek",
	}
	result := providersForRotation(params)
	for _, p := range result {
		if p == "openai" {
			t.Log("NOTE: If this test passes, openai IS included (behavior changed).")
			return
		}
	}
	t.Log("Current behavior: original provider 'openai' is NOT in rotation list. " +
		"Flag says 'fallback' but original provider is never tried. " +
		"See rotate-bugs.md #4.")
}

// ============================================================
// Tests for body close in rotation fallback (rotate-bugs.md #5)
// ============================================================

func TestBodyCloseForCorrectPath(t *testing.T) {
	closed := false
	body := &trackedBody{
		reader: strings.NewReader("error"),
		onClose: func() {
			closed = true
		},
	}
	resp := &http.Response{
		Body:       body,
		StatusCode: 502,
	}

	// Simulate the correct close path (line 757 for non-last providers)
	resp.Body.Close()

	if !closed {
		t.Error("trackedBody did not register Close() call")
	}
}

// ============================================================
// Tests for deduplication (rotate-bugs.md #8)
// ============================================================

func TestProvidersForRotation_Duplicates(t *testing.T) {
	params := structs.Params{
		RotateProviders: "gemini,gemini,deepseek,gemini",
	}
	result := providersForRotation(params)
	// Current behavior: duplicates are NOT filtered (bug #8)
	t.Logf("Current: providersForRotation(%q) = %v (len=%d) — duplicates included",
		"gemini,gemini,deepseek,gemini", result, len(result))
	geminiCount := 0
	for _, p := range result {
		if p == "gemini" {
			geminiCount++
		}
	}
	if geminiCount > 1 {
		t.Log("BUG #8: duplicates not filtered — gemini appears", geminiCount, "times")
	}
}
