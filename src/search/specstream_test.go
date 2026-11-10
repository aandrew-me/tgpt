package search

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSpecStreamRequestURL(t *testing.T) {
	got := specstreamRequestURL(specstreamEndpoint, SearchParams{Query: "docker tutorial", NumResults: 3, SiteFilter: "reddit.com"})
	want := "https://api.specstream.ai/v1/web-search?num=3&passages=true&q=site%3Areddit.com+docker+tutorial"
	if got != want {
		t.Errorf("unexpected URL:\n got %s\nwant %s", got, want)
	}
	if got := specstreamRequestURL(specstreamEndpoint, SearchParams{Query: "q", NumResults: 500}); !strings.Contains(got, "num=10&") {
		t.Errorf("expected num capped at 10, got %s", got)
	}
	if got := specstreamRequestURL(specstreamEndpoint, SearchParams{Query: "q"}); !strings.Contains(got, "num=5&") || strings.Contains(got, "site%3A") {
		t.Errorf("expected 5 results by default and no site filter, got %s", got)
	}
}

func TestParseSpecStreamResponse(t *testing.T) {
	body := []byte(`{"data": {
		"request_id": "req_1", "query": "go error handling", "kind": "fact", "columns": [], "answer": null, "rows": [],
		"sources": [
			{"url": "https://go.dev/blog/error-handling-and-go", "title": "Error handling and Go", "site": "go.dev", "snippet": "Errors are values.", "cited": true, "published": "2011-07-12"},
			{"url": "not a url", "title": "Broken link", "site": "", "snippet": "should be skipped", "cited": false},
			{"url": "https://go.dev/doc/effective_go#errors", "title": "Effective Go", "site": "go.dev", "snippet": "Library routines must often return some sort of error indication.", "cited": false, "published": null}
		],
		"passages": [
			{"url": "https://go.dev/blog/error-handling-and-go", "title": "Error handling and Go", "text": "Go code uses error values to indicate an abnormal state."}
		],
		"handoff": null, "took_ms": 1200, "price_usd": 0.005
	}}`)

	results, err := parseSpecStreamResponse(body, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (invalid URL skipped), got %d", len(results))
	}
	if results[0].URL != "https://go.dev/blog/error-handling-and-go" || results[0].Snippet != "(published 2011-07-12) Errors are values." {
		t.Errorf("unexpected first result: %+v", results[0])
	}
	if results[0].Content != "Go code uses error values to indicate an abnormal state." {
		t.Errorf("unexpected first content: %q", results[0].Content)
	}
	if results[1].Snippet != "Library routines must often return some sort of error indication." || results[1].Content != "" {
		t.Errorf("unexpected second result: %+v", results[1])
	}

	if _, err := parseSpecStreamResponse([]byte(`not json`), false); err == nil {
		t.Error("expected an error for invalid JSON")
	}
}

func TestSpecStreamSearchHTTP(t *testing.T) {
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/v1/web-search" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected Authorization header %q", r.Header.Get("Authorization"))
		}
		if q := r.URL.Query(); q.Get("q") != "site:go.dev release notes" || q.Get("num") != "2" || q.Get("passages") != "true" {
			t.Errorf("unexpected query %v", q)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status != http.StatusOK {
			fmt.Fprint(w, `{"error": {"code": "invalid_token", "message": "invalid key"}}`)
			return
		}
		fmt.Fprint(w, `{"data": {"sources": [{"url": "https://go.dev/doc/devel/release", "title": "Release History", "site": "go.dev", "snippet": "go1.26 released", "cited": true}]}}`)
	}))
	defer server.Close()

	params := SearchParams{Query: "release notes", NumResults: 2, SiteFilter: "go.dev"}
	results, err := specstreamSearch(params, "test-key", server.URL+"/v1/web-search", server.Client(), false)
	if err != nil || len(results) != 1 || results[0].Title != "Release History" {
		t.Fatalf("results=%+v, error=%v", results, err)
	}

	status = http.StatusUnauthorized
	_, err = specstreamSearch(params, "test-key", server.URL+"/v1/web-search", server.Client(), false)
	if err == nil || !strings.Contains(err.Error(), "status 401: invalid key (invalid_token)") {
		t.Fatalf("expected the API error message, got %v", err)
	}
}

func TestSpecStreamRequiresKey(t *testing.T) {
	t.Setenv("SPECSTREAM_API_KEY", "")
	if _, err := PerformSpecStreamSearch(SearchParams{Query: "q", NumResults: 1}, false); err == nil || !strings.Contains(err.Error(), "SPECSTREAM_API_KEY") {
		t.Fatalf("SpecStream missing-key error: %v", err)
	}
}
