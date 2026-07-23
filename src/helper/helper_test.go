package helper

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

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


