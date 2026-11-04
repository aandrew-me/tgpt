package search

import "testing"

func TestParseSerpingapiResponse(t *testing.T) {
	body := []byte(`{
		"searchParameters": {"q": "go error handling", "type": "search"},
		"organic": [
			{"title": "Error handling in Go", "link": "https://go.dev/blog/error-handling-and-go", "snippet": "Errors are values.", "position": 1},
			{"title": "Broken link", "link": "not a url", "snippet": "should be skipped", "position": 2},
			{"title": "Effective Go", "link": "https://go.dev/doc/effective_go#errors", "snippet": "Library routines must often return some sort of error indication.", "position": 3}
		],
		"answerBox": {"title": "ignored"}
	}`)

	results, err := parseSerpingapiResponse(body, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results (invalid URL skipped), got %d", len(results))
	}

	if results[0].Title != "Error handling in Go" || results[0].URL != "https://go.dev/blog/error-handling-and-go" || results[0].Snippet != "Errors are values." {
		t.Errorf("unexpected first result: %+v", results[0])
	}

	if results[1].URL != "https://go.dev/doc/effective_go#errors" {
		t.Errorf("unexpected second result: %+v", results[1])
	}

	if results[0].Content != "" {
		t.Errorf("expected empty Content before extraction, got %q", results[0].Content)
	}
}

func TestParseSerpingapiResponseEmptyAndInvalid(t *testing.T) {
	results, err := parseSerpingapiResponse([]byte(`{"organic": []}`), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}

	if _, err := parseSerpingapiResponse([]byte(`not json`), false); err == nil {
		t.Errorf("expected error for invalid JSON")
	}
}

func TestSerpingapiQuery(t *testing.T) {
	if got := serpingapiQuery(SearchParams{Query: "docker tutorial"}); got != "docker tutorial" {
		t.Errorf("unexpected query without site filter: %q", got)
	}

	if got := serpingapiQuery(SearchParams{Query: "docker tutorial", SiteFilter: "reddit.com"}); got != "site:reddit.com docker tutorial" {
		t.Errorf("unexpected query with site filter: %q", got)
	}
}

func TestParseSerplyResponse(t *testing.T) {
	body := []byte(`{
		"results": [
			{"title": "Error handling in Go", "link": "https://go.dev/blog/error-handling-and-go", "description": "Errors are values.", "position": 1},
			{"title": "Broken link", "link": "not a url", "description": "should be skipped", "position": 2},
			{"title": "Effective Go", "link": "https://go.dev/doc/effective_go#errors", "description": "Library routines must often return some sort of error indication.", "position": 3}
		],
		"related_searches": [{"query": "ignored"}]
	}`)

	results, err := parseSerplyResponse(body, 5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results (invalid URL skipped), got %d", len(results))
	}

	if results[0].Title != "Error handling in Go" || results[0].URL != "https://go.dev/blog/error-handling-and-go" || results[0].Snippet != "Errors are values." {
		t.Errorf("unexpected first result: %+v", results[0])
	}

	if results[1].URL != "https://go.dev/doc/effective_go#errors" {
		t.Errorf("unexpected second result: %+v", results[1])
	}

	if results[0].Content != "" {
		t.Errorf("expected empty Content before extraction, got %q", results[0].Content)
	}
}

func TestParseSerplyResponseEmptyAndInvalid(t *testing.T) {
	results, err := parseSerplyResponse([]byte(`{"results": []}`), 5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected no results, got %d", len(results))
	}

	if _, err := parseSerplyResponse([]byte(`not json`), 5, false); err == nil {
		t.Errorf("expected error for invalid JSON")
	}
}

// Serply treats num as an approximate cap rather than an exact count, so the
// parser trims any surplus rows the API returns.
func TestParseSerplyResponseTrimsToNumResults(t *testing.T) {
	body := []byte(`{
		"results": [
			{"title": "One", "link": "https://example.com/1", "description": "a"},
			{"title": "Two", "link": "https://example.com/2", "description": "b"},
			{"title": "Three", "link": "https://example.com/3", "description": "c"}
		]
	}`)

	results, err := parseSerplyResponse(body, 2, false)
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

func TestSerplyQuery(t *testing.T) {
	if got := serplyQuery(SearchParams{Query: "docker tutorial"}); got != "docker tutorial" {
		t.Errorf("unexpected query without site filter: %q", got)
	}

	if got := serplyQuery(SearchParams{Query: "docker tutorial", SiteFilter: "reddit.com"}); got != "site:reddit.com docker tutorial" {
		t.Errorf("unexpected query with site filter: %q", got)
	}
}

func TestParseKeenableResponse(t *testing.T) {
	body := []byte(`{
		"query": "go error handling",
		"mode": "pro",
		"results": [
			{"title": "Error handling and Go", "url": "https://go.dev/blog/error-handling-and-go", "description": "", "snippet": "Errors are values.", "published_at": "2011-07-12T00:00:00Z"},
			{"title": "Broken link", "url": "not a url", "description": "", "snippet": "should be skipped"},
			{"title": "Effective Go", "url": "https://go.dev/doc/effective_go#errors", "description": "", "snippet": "Library routines must often return some sort of error indication."}
		]
	}`)

	results, err := parseKeenableResponse(body, 5, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results (invalid URL skipped), got %d", len(results))
	}
	if results[0].Title != "Error handling and Go" || results[0].URL != "https://go.dev/blog/error-handling-and-go" {
		t.Errorf("unexpected first result: %+v", results[0])
	}
	if results[0].Snippet != "(published 2011-07-12T00:00:00Z) Errors are values." {
		t.Errorf("unexpected first snippet: %q", results[0].Snippet)
	}
	if results[1].Snippet != "Library routines must often return some sort of error indication." {
		t.Errorf("unexpected second snippet: %q", results[1].Snippet)
	}

	trimmed, err := parseKeenableResponse(body, 1, false)
	if err != nil || len(trimmed) != 1 {
		t.Fatalf("expected 1 result when trimmed, got %d (%v)", len(trimmed), err)
	}

	if _, err := parseKeenableResponse([]byte(`not json`), 5, false); err == nil {
		t.Error("expected an error for invalid JSON")
	}
}

func TestKeenableEndpointAndBody(t *testing.T) {
	endpoint, headers := keenableEndpoint("")
	if endpoint != "https://api.keenable.ai/v1/search/public" || headers["X-Keenable-Title"] != "tgpt" || headers["X-API-Key"] != "" {
		t.Errorf("unexpected keyless endpoint %q / headers %v", endpoint, headers)
	}
	endpoint, headers = keenableEndpoint("keen_abc")
	if endpoint != "https://api.keenable.ai/v1/search" || headers["X-API-Key"] != "keen_abc" || headers["X-Keenable-Title"] != "" {
		t.Errorf("unexpected keyed endpoint %q / headers %v", endpoint, headers)
	}

	body := keenableRequestBody(SearchParams{Query: "docker tutorial", NumResults: 3, SiteFilter: "reddit.com"})
	if body["query"] != "docker tutorial" || body["max_results"] != 3 || body["site"] != "reddit.com" {
		t.Errorf("unexpected body: %v", body)
	}
	body = keenableRequestBody(SearchParams{Query: "q", NumResults: 500})
	if body["max_results"] != 50 {
		t.Errorf("expected max_results capped at 50, got %v", body["max_results"])
	}
	if _, ok := body["site"]; ok {
		t.Error("expected no site field without a site filter")
	}
	if keenableRequestBody(SearchParams{Query: "q"})["max_results"] != 5 {
		t.Error("expected 5 results by default")
	}
}

func TestParseSofyaResponse(t *testing.T) {
	body := []byte(`{
		"query": "go error handling",
		"answer": null,
		"results": [
			{"title": "Error handling and Go", "url": "https://go.dev/blog/error-handling-and-go", "description": "Errors are values.", "content": "Go code uses error values to indicate an abnormal state.", "fetched": true, "published_date": "2011-07-12"},
			{"title": "Broken link", "url": "not a url", "description": "should be skipped", "content": ""},
			{"title": "Effective Go", "url": "https://go.dev/doc/effective_go#errors", "description": "Library routines must often return some sort of error indication.", "content": "", "fetched": false}
		],
		"credits_used": 2
	}`)

	results, err := parseSofyaResponse(body, false)
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
	if results[1].Content != "" {
		t.Errorf("expected empty content for an unfetched page, got %q", results[1].Content)
	}

	if _, err := parseSofyaResponse([]byte(`not json`), false); err == nil {
		t.Error("expected an error for invalid JSON")
	}
}

func TestSofyaRequestBody(t *testing.T) {
	body := sofyaRequestBody(SearchParams{Query: "docker tutorial", NumResults: 3, SiteFilter: "reddit.com"})
	if body["query"] != "docker tutorial" || body["max_results"] != 3 || body["search_depth"] != "basic" {
		t.Errorf("unexpected body: %v", body)
	}
	if domains, _ := body["include_domains"].([]string); len(domains) != 1 || domains[0] != "reddit.com" {
		t.Errorf("unexpected include_domains: %v", body["include_domains"])
	}
	body = sofyaRequestBody(SearchParams{Query: "q", NumResults: 500})
	if body["max_results"] != 20 {
		t.Errorf("expected max_results capped at 20, got %v", body["max_results"])
	}
	if _, ok := body["include_domains"]; ok {
		t.Error("expected no include_domains without a site filter")
	}
	if sofyaRequestBody(SearchParams{Query: "q"})["max_results"] != 5 {
		t.Error("expected 5 results by default")
	}
}
