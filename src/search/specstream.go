package search

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"github.com/aandrew-me/tgpt/v2/src/client"
)

const specstreamEndpoint = "https://api.specstream.ai/v1/web-search"

// SpecStreamSearchResponse holds the parts of a SpecStream Search API reply
// that tgpt uses. With passages=true the reply also carries page text.
type SpecStreamSearchResponse struct {
	Data struct {
		Sources []struct {
			Title     string `json:"title"`
			URL       string `json:"url"`
			Snippet   string `json:"snippet"`
			Published string `json:"published"`
		} `json:"sources"`
		Passages []struct {
			URL  string `json:"url"`
			Text string `json:"text"`
		} `json:"passages"`
	} `json:"data"`
}

// specstreamRequestURL builds the GET address of a SpecStream search. The
// site filter goes into the query as a site: operator, and passages asks for
// the page text, so no content extractor is needed.
func specstreamRequestURL(endpoint string, params SearchParams) string {
	numResults := params.NumResults
	if numResults <= 0 {
		numResults = 5
	}
	// A call of up to 10 results is charged at the base price.
	if numResults > 10 {
		numResults = 10
	}
	query := params.Query
	if params.SiteFilter != "" {
		query = "site:" + params.SiteFilter + " " + query
	}
	values := url.Values{}
	values.Set("q", query)
	values.Set("num", strconv.Itoa(numResults))
	values.Set("passages", "true")
	return endpoint + "?" + values.Encode()
}

// specstreamSearch performs the actual SpecStream search API call
func specstreamSearch(params SearchParams, apiKey, endpoint string, httpClient *http.Client, verbose bool) ([]SearchResult, error) {
	requestURL := specstreamRequestURL(endpoint, params)
	if verbose {
		fmt.Printf("Calling SpecStream: GET %s\n", requestURL)
	}

	req, err := http.NewRequest("GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("User-Agent", "TGPT/2.11.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read search response: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		// SpecStream errors carry a code and a message, for example
		// invalid_token or insufficient_balance; show the message.
		var apiErr struct {
			Error struct {
				Code    string `json:"code"`
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("search API returned status %d: %s (%s)", resp.StatusCode, apiErr.Error.Message, apiErr.Error.Code)
		}
		return nil, fmt.Errorf("search API returned status %d: %s", resp.StatusCode, string(body))
	}

	return parseSpecStreamResponse(body, verbose)
}

// parseSpecStreamResponse converts a SpecStream JSON response into search
// results. Passage text is attached to the source with the same URL.
func parseSpecStreamResponse(body []byte, verbose bool) ([]SearchResult, error) {
	var searchResp SpecStreamSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %v", err)
	}

	passages := map[string][]string{}
	for _, passage := range searchResp.Data.Passages {
		passages[passage.URL] = append(passages[passage.URL], passage.Text)
	}

	var results []SearchResult
	for _, item := range searchResp.Data.Sources {
		if _, err := url.ParseRequestURI(item.URL); err != nil {
			if verbose {
				fmt.Printf("Warning: Skipping invalid URL: %s\n", item.URL)
			}
			continue
		}
		snippet := item.Snippet
		if item.Published != "" {
			snippet = "(published " + item.Published + ") " + snippet
		}
		content := strings.Join(passages[item.URL], "\n")
		if r := []rune(content); len(r) > maxContentPerURL {
			content = string(r[:maxContentPerURL]) + "..."
		}
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.URL,
			Snippet: snippet,
			Content: content,
		})
	}

	return results, nil
}

// PerformSpecStreamSearch searches with the SpecStream Search API and formats
// the results for the model. It needs SPECSTREAM_API_KEY but no content
// extractor: the reply carries page passages.
func PerformSpecStreamSearch(params SearchParams, verbose bool) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv("SPECSTREAM_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("missing required environment variable: SPECSTREAM_API_KEY must be set. Please check SEARCH_SETUP.md for configuration instructions")
	}

	httpClient := client.NewStandardHTTPClient(60)
	defer httpClient.CloseIdleConnections()
	results, err := specstreamSearch(params, apiKey, specstreamEndpoint, httpClient, verbose)
	if err != nil {
		return "", fmt.Errorf("search failed: %v", err)
	}
	return formatResultsForAI(results, params.Query), nil
}
