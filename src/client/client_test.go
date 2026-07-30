package client

import (
	"os"
	"testing"
)

func TestGetProxyURLFromEnv(t *testing.T) {
	os.Setenv("HTTP_PROXY", "http://127.0.0.1:8080")
	defer os.Unsetenv("HTTP_PROXY")

	proxy := GetProxyURL()
	if proxy != "http://127.0.0.1:8080" {
		t.Fatalf("expected http://127.0.0.1:8080, got %q", proxy)
	}
}

func TestNewStandardHTTPClient(t *testing.T) {
	c := NewStandardHTTPClient(30)
	if c == nil {
		t.Fatal("expected non-nil http.Client")
	}
	if c.Timeout.Seconds() != 30 {
		t.Fatalf("expected timeout 30s, got %v", c.Timeout)
	}
}
