package search

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSearchapiQuery(t *testing.T) {
	if got := searchapiQuery(SearchParams{Query: "docker tutorial"}); got != "docker tutorial" {
		t.Errorf("unexpected query without site filter: %q", got)
	}

	if got := searchapiQuery(SearchParams{Query: "docker tutorial", SiteFilter: "reddit.com"}); got != "site:reddit.com docker tutorial" {
		t.Errorf("unexpected query with site filter: %q", got)
	}
}

func TestParseSearchapiResponse(t *testing.T) {
	body := []byte(`{
		"search_metadata": {"id": "search_1", "status": "Success"},
		"search_parameters": {"engine": "google", "q": "go error handling", "num": 5},
		"organic_results": [
			{"position": 1, "title": "Working with Errors in Go 1.13", "link": "https://go.dev/blog/go1.13-errors", "source": "The Go Programming Language", "domain": "go.dev", "snippet": "Go 1.13 introduces new features to the errors and fmt standard library packages.", "date": "Oct 17, 2019"},
			{"position": 2, "title": "Broken link", "link": "not a url", "snippet": "should be skipped"},
			{"position": 3, "title": "Effective Go", "link": "https://go.dev/doc/effective_go#errors", "source": "The Go Programming Language", "domain": "go.dev", "snippet": "Library routines must often return some sort of error indication."}
		],
		"related_searches": [{"query": "ignored"}],
		"pagination": {"current": 1}
	}`)

	results, err := parseSearchapiResponse(body, 5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results (invalid URL skipped), got %d", len(results))
	}

	if results[0].Title != "Working with Errors in Go 1.13" || results[0].URL != "https://go.dev/blog/go1.13-errors" {
		t.Errorf("unexpected first result: %+v", results[0])
	}
	if results[0].Snippet != "(published Oct 17, 2019) Go 1.13 introduces new features to the errors and fmt standard library packages." {
		t.Errorf("unexpected first snippet: %q", results[0].Snippet)
	}

	if results[1].URL != "https://go.dev/doc/effective_go#errors" || results[1].Snippet != "Library routines must often return some sort of error indication." {
		t.Errorf("unexpected second result: %+v", results[1])
	}

	if results[0].Content != "" {
		t.Errorf("expected empty Content before extraction, got %q", results[0].Content)
	}
}

func TestParseSearchapiResponseEmptyAndInvalid(t *testing.T) {
	results, err := parseSearchapiResponse([]byte(`{"organic_results": []}`), 5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}

	// A page with no organic section at all (for example only a knowledge graph).
	results, err = parseSearchapiResponse([]byte(`{"search_metadata": {"status": "Success"}}`), 5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results without organic_results, got %d", len(results))
	}

	if _, err := parseSearchapiResponse([]byte(`not json`), 5, false); err == nil {
		t.Errorf("expected error for invalid JSON")
	}
}

// Google does not always honor num and returns a full page of rows, so the
// parser trims any surplus rows to the requested count.
func TestParseSearchapiResponseTrimsToNumResults(t *testing.T) {
	body := []byte(`{
		"organic_results": [
			{"position": 1, "title": "One", "link": "https://example.com/1", "snippet": "a"},
			{"position": 2, "title": "Two", "link": "https://example.com/2", "snippet": "b"},
			{"position": 3, "title": "Three", "link": "https://example.com/3", "snippet": "c"}
		]
	}`)

	results, err := parseSearchapiResponse(body, 2, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results after trimming, got %d", len(results))
	}
	if results[1].URL != "https://example.com/2" {
		t.Errorf("unexpected second result: %+v", results[1])
	}
}

func TestSearchapiSearchHTTP(t *testing.T) {
	status := http.StatusOK
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" || r.URL.Path != "/api/v1/search" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("unexpected Authorization header %q", r.Header.Get("Authorization"))
		}
		q := r.URL.Query()
		if q.Get("engine") != "google" || q.Get("q") != "site:go.dev release notes" || q.Get("num") != "2" {
			t.Errorf("unexpected query %v", q)
		}
		if q.Has("api_key") || strings.Contains(r.URL.RawQuery, "test-key") {
			t.Errorf("API key must not be sent in the URL: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		if status != http.StatusOK {
			fmt.Fprint(w, `{"error": "Invalid API key."}`)
			return
		}
		fmt.Fprint(w, `{"organic_results": [{"position": 1, "title": "Release History", "link": "https://go.dev/doc/devel/release", "domain": "go.dev", "snippet": "go1.26 released"}]}`)
	}))
	defer server.Close()

	params := SearchParams{Query: "release notes", NumResults: 2, SiteFilter: "go.dev"}
	results, err := searchapiSearch(params, "test-key", server.URL+"/api/v1/search", server.Client(), false)
	if err != nil || len(results) != 1 || results[0].Title != "Release History" {
		t.Fatalf("results=%+v, error=%v", results, err)
	}

	status = http.StatusUnauthorized
	_, err = searchapiSearch(params, "test-key", server.URL+"/api/v1/search", server.Client(), false)
	if err == nil || !strings.Contains(err.Error(), "status 401: Invalid API key.") {
		t.Fatalf("expected the API error message, got %v", err)
	}
}

func TestSearchapiRequiresKey(t *testing.T) {
	t.Setenv("SEARCHAPI_API_KEY", "")
	if _, err := PerformSearchapiSearch(SearchParams{Query: "q", NumResults: 1}, false); err == nil || !strings.Contains(err.Error(), "SEARCHAPI_API_KEY") {
		t.Fatalf("SearchApi missing-key error: %v", err)
	}
}
