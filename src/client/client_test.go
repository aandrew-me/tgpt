package client

import (
	"net/http"
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

	defaultClient := NewStandardHTTPClient()
	if defaultClient.Timeout.Seconds() != 600 {
		t.Fatalf("expected default timeout 600s, got %v", defaultClient.Timeout)
	}

	tr, ok := defaultClient.Transport.(*http.Transport)
	if !ok {
		t.Fatal("expected *http.Transport")
	}
	if tr.DialContext == nil {
		t.Fatal("expected DialContext to be set")
	}
	if tr.ResponseHeaderTimeout != 0 {
		t.Fatalf("expected ResponseHeaderTimeout to be 0, got %v", tr.ResponseHeaderTimeout)
	}
}

func TestNewClient(t *testing.T) {
	c, err := NewClient()
	if err != nil {
		t.Fatalf("unexpected error creating tls client: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil tls client")
	}
}
