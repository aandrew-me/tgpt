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
