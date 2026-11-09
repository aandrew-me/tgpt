package magicstudio

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	http "github.com/bogdanfinn/fhttp"
)

func TestSaveImageResponse(t *testing.T) {
	t.Chdir(t.TempDir())
	for _, tc := range []struct {
		name        string
		status      int
		contentType string
		body        string
		wantError   string
	}{
		{"http error", 429, "application/json", `{"error":"rate limited"}`, "429"},
		{"http error with image header", 500, "image/jpeg", "server error", "500"},
		{"unexpected content", 200, "application/json", `{"error":"generation failed"}`, "application/json"},
		{"missing content type", 200, "", "unexpected response", "Content-Type"},
		{"jpeg", 200, "image/jpeg", "image bytes", ""},
		{"jpg", 200, "image/jpg", "image bytes", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp := &http.Response{StatusCode: tc.status, Header: http.Header{"Content-Type": {tc.contentType}}, Body: io.NopCloser(strings.NewReader(tc.body))}
			filename, err := saveImageResponse(resp, "test")
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) || !strings.Contains(err.Error(), tc.body) {
					t.Fatalf("expected diagnostic containing %q and response body; filename=%q, error=%v", tc.wantError, filename, err)
				}
				if filename != "" {
					t.Fatalf("unexpected filename %q", filename)
				}
				files, _ := filepath.Glob("magic_*.jpg")
				if len(files) != 0 {
					t.Fatalf("error response saved: %v", files)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			defer os.Remove(filename)
			data, err := os.ReadFile(filename)
			if err != nil || string(data) != tc.body {
				t.Fatalf("saved image: %q, %v", data, err)
			}
		})
	}
}
