package search

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/aandrew-me/tgpt/v2/src/client"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

const parallelEndpoint = "https://mcp.parallel.ai/mcp"
const parallelUserAgent = "tgpt/parallel-search (+https://github.com/aandrew-me/tgpt)"

// Set the header at the HTTP boundary, including SDK notifications and session
// teardown, which do not all use the SDK's per-request header options.
type parallelTransport struct{ base http.RoundTripper }

func (t parallelTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", parallelUserAgent)
	return t.base.RoundTrip(req)
}

// PerformParallelSearch uses anonymous Search MCP, whose server-managed default
// is Fast mode. It returns source excerpts without a separate content extractor.
func PerformParallelSearch(ctx context.Context, params SearchParams, verbose bool) (string, error) {
	httpClient := client.NewStandardHTTPClient(60)
	defer httpClient.CloseIdleConnections()
	httpClient.Transport = parallelTransport{base: httpClient.Transport}
	return performParallelSearch(ctx, params, verbose, parallelEndpoint, httpClient)
}

func performParallelSearch(ctx context.Context, params SearchParams, verbose bool, endpoint string, httpClient *http.Client) (string, error) {
	if strings.TrimSpace(params.Query) == "" {
		return "", fmt.Errorf("query parameter is required")
	}
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	c, err := mcpclient.NewStreamableHttpClient(endpoint, transport.WithHTTPBasicClient(httpClient))
	if err != nil {
		return "", err
	}
	defer c.Close()
	if err := c.Start(ctx); err != nil {
		return "", fmt.Errorf("Parallel MCP start: %w", err)
	}
	init := mcp.InitializeRequest{}
	init.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	init.Params.ClientInfo = mcp.Implementation{Name: "tgpt", Version: "1.0.0"}
	if _, err := c.Initialize(ctx, init); err != nil {
		return "", fmt.Errorf("Parallel MCP initialize: %w", err)
	}
	available, err := c.ListTools(ctx, mcp.ListToolsRequest{})
	if err != nil {
		return "", fmt.Errorf("Parallel MCP discovery: %w", err)
	}
	found := false
	for _, tool := range available.Tools {
		if tool.Name == "web_search" {
			found = true
			break
		}
	}
	if !found {
		return "", fmt.Errorf("Parallel MCP does not expose web_search")
	}
	query := params.Query
	if params.SiteFilter != "" {
		query = "site:" + params.SiteFilter + " " + query
	}
	call := mcp.CallToolRequest{}
	call.Params.Name = "web_search"
	// Anonymous MCP does not expose a result-count setting. Limit locally.
	call.Params.Arguments = map[string]any{"objective": params.Query, "search_queries": []string{query}}
	if verbose {
		fmt.Printf("[Parallel MCP] Searching: %q\n", query)
	}
	result, err := c.CallTool(ctx, call)
	if err != nil {
		return "", fmt.Errorf("Parallel search: %w", err)
	}
	if result.IsError {
		return "", fmt.Errorf("Parallel search: %s", parallelText(result))
	}
	data := result.RawStructuredContent
	if len(data) == 0 && result.StructuredContent != nil {
		data, err = json.Marshal(result.StructuredContent)
		if err != nil {
			return "", err
		}
	}
	if len(data) == 0 {
		data = []byte(parallelText(result))
	}
	return formatParallelResults(data, params.NumResults)
}

func parallelText(result *mcp.CallToolResult) string {
	var parts []string
	for _, content := range result.Content {
		if text, ok := content.(mcp.TextContent); ok {
			parts = append(parts, text.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func formatParallelResults(data []byte, limit int) (string, error) {
	var response struct {
		Results []struct {
			Title    string   `json:"title"`
			URL      string   `json:"url"`
			Excerpts []string `json:"excerpts"`
		} `json:"results"`
		Warnings []struct {
			Message string `json:"message"`
		} `json:"warnings"`
	}
	if err := json.Unmarshal(data, &response); err != nil {
		return "", fmt.Errorf("invalid Parallel search response: %w", err)
	}
	if limit <= 0 {
		limit = 5
	}
	if limit > len(response.Results) {
		limit = len(response.Results)
	}
	var out strings.Builder
	for i, result := range response.Results[:limit] {
		excerpt := []rune(strings.Join(result.Excerpts, "\n"))
		if len(excerpt) > maxContentPerURL {
			excerpt = excerpt[:maxContentPerURL]
		}
		fmt.Fprintf(&out, "%d. %s\nURL: %s\n%s\n\n", i+1, result.Title, result.URL, string(excerpt))
	}
	if limit == 0 {
		out.WriteString("No search results found.\n")
	}
	for _, warning := range response.Warnings {
		fmt.Fprintf(&out, "Warning: %s\n", warning.Message)
	}
	return out.String(), nil
}
