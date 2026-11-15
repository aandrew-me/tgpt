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

const searchapiEndpoint = "https://www.searchapi.io/api/v1/search"

// SearchapiSearchResponse holds the part of a SearchApi Google reply that
// tgpt uses: the organic rows. The other sections (knowledge graph, related
// searches, pagination) are ignored.
type SearchapiSearchResponse struct {
	OrganicResults []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
		Date    string `json:"date"`
	} `json:"organic_results"`
}

// searchapiQuery builds the q parameter sent to SearchApi. Like serpingapi and
// Serply, the API has no separate site filter parameter, so the filter is
// expressed as a site: operator.
func searchapiQuery(params SearchParams) string {
	if params.SiteFilter != "" {
		return "site:" + params.SiteFilter + " " + params.Query
	}
	return params.Query
}

// searchapiSearch performs the actual SearchApi search API call. The key is
// sent as a bearer token, never in the URL.
func searchapiSearch(params SearchParams, apiKey, endpoint string, httpClient *http.Client, verbose bool) ([]SearchResult, error) {
	numResults := params.NumResults
	if numResults <= 0 {
		numResults = 5
	}

	query := url.Values{}
	query.Set("engine", "google")
	query.Set("q", searchapiQuery(params))
	query.Set("num", strconv.Itoa(numResults))
	requestURL := endpoint + "?" + query.Encode()

	if verbose {
		fmt.Printf("Calling SearchApi: GET %s\n", requestURL)
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
		// SearchApi errors are {"error": "message"}; show the message.
		var apiErr struct {
			Error string `json:"error"`
		}
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Error != "" {
			return nil, fmt.Errorf("search API returned status %d: %s", resp.StatusCode, apiErr.Error)
		}
		return nil, fmt.Errorf("search API returned status %d: %s", resp.StatusCode, string(body))
	}

	return parseSearchapiResponse(body, numResults, verbose)
}

// parseSearchapiResponse converts a SearchApi JSON response into search
// results. Google does not always honor num and returns a full page of
// organic rows, so the rows are trimmed to the requested number here. The
// publish date, when Google shows one, is added in front of the snippet.
func parseSearchapiResponse(body []byte, numResults int, verbose bool) ([]SearchResult, error) {
	var searchResp SearchapiSearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %v", err)
	}

	var results []SearchResult
	for _, item := range searchResp.OrganicResults {
		if numResults > 0 && len(results) >= numResults {
			break
		}
		if _, err := url.ParseRequestURI(item.Link); err != nil {
			if verbose {
				fmt.Printf("Warning: Skipping invalid URL: %s\n", item.Link)
			}
			continue
		}
		snippet := item.Snippet
		if item.Date != "" {
			snippet = "(published " + item.Date + ") " + snippet
		}
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: snippet,
		})
	}

	return results, nil
}

// PerformSearchapiSearch searches Google through SearchApi and formats the
// results for the model. It needs SEARCHAPI_API_KEY. The reply carries
// snippets only, so page content is fetched with is-fast as for the other
// Google providers.
func PerformSearchapiSearch(params SearchParams, verbose bool) (string, error) {
	apiKey := strings.TrimSpace(os.Getenv("SEARCHAPI_API_KEY"))
	if apiKey == "" {
		return "", fmt.Errorf("missing required environment variable: SEARCHAPI_API_KEY must be set. Please check SEARCH_SETUP.md for configuration instructions")
	}

	httpClient := client.NewStandardHTTPClient(60)
	defer httpClient.CloseIdleConnections()
	results, err := searchapiSearch(params, apiKey, searchapiEndpoint, httpClient, verbose)
	if err != nil {
		return "", fmt.Errorf("search failed: %v", err)
	}

	extractContentForResults(results, verbose)

	return formatResultsForAI(results, params.Query), nil
}
