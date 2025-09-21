package search

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/providers"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	http "github.com/bogdanfinn/fhttp"
)

const (
	maxContentPerURL         = 8000             // Maximum content per search result
	maxTotalContent          = 100000           // Maximum total content for AI processing
	extractionTimeout        = 15 * time.Second // Timeout for content extraction
	maxConcurrentExtractions = 5                // Maximum concurrent content extractions
)

// SearchParams represents the parameters extracted by AI for search
type SearchParams struct {
	Query      string `json:"query"`
	NumResults int    `json:"num_results"`
	SiteFilter string `json:"site_filter,omitempty"`
}

// SearchResult represents a single search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
	Content string `json:"content,omitempty"`
}

// GoogleSearchResponse represents the response from Google Custom Search API
type GoogleSearchResponse struct {
	Items []struct {
		Title   string `json:"title"`
		Link    string `json:"link"`
		Snippet string `json:"snippet"`
	} `json:"items"`
}

// PerformSearch executes the complete search workflow
func PerformSearch(userQuery string, verbose bool) (string, error) {
	// Get API credentials from environment
	apiKey := os.Getenv("TGPT_GOOGLE_API_KEY")
	searchEngineID := os.Getenv("TGPT_GOOGLE_SEARCH_ENGINE_ID")

	if apiKey == "" || searchEngineID == "" {
		return "", fmt.Errorf("missing required environment variables: TGPT_GOOGLE_API_KEY and TGPT_GOOGLE_SEARCH_ENGINE_ID must be set. Please check SEARCH_SETUP.md for configuration instructions")
	}

	// Extract search parameters using AI (this would be called from helper.go)
	// For now, we'll use simple defaults
	params := SearchParams{
		Query:      userQuery,
		NumResults: 3,
	}

	// Perform Google search
	results, err := googleSearch(params, apiKey, searchEngineID, verbose)
	if err != nil {
		return "", fmt.Errorf("search failed: %v", err)
	}

	// Extract content from each result
	for i := range results {
		if verbose {
			fmt.Printf("Extracting content from result %d: %s\n", i+1, results[i].URL)
		}
		content, err := extractContent(results[i].URL)
		if err != nil {
			// Log error but continue with other results
			if verbose {
				fmt.Fprintf(os.Stderr, "Failed to extract content from %s: %v\n", results[i].URL, err)
			}
			continue
		}
		if verbose {
			fmt.Printf("Successfully extracted %d characters from result %d\n", len(content), i+1)
		}
		results[i].Content = content
	}

	// Format results for AI synthesis
	return formatResultsForAI(results, userQuery), nil
}

// googleSearch performs the actual Google Custom Search API call
func googleSearch(params SearchParams, apiKey, searchEngineID string, verbose bool) ([]SearchResult, error) {
	baseURL := "https://www.googleapis.com/customsearch/v1"

	// Build query parameters
	queryParams := url.Values{}
	queryParams.Set("key", apiKey)
	queryParams.Set("cx", searchEngineID)
	queryParams.Set("q", params.Query)
	queryParams.Set("num", fmt.Sprintf("%d", params.NumResults))

	if params.SiteFilter != "" {
		queryParams.Set("siteSearch", params.SiteFilter)
	}

	searchURL := baseURL + "?" + queryParams.Encode()
	if verbose {
		fmt.Printf("Calling Google API: %s\n", searchURL)
	}

	// Create HTTP client
	httpClient, err := client.NewClient()
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %v", err)
	}

	// Make request
	req, err := http.NewRequest("GET", searchURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("User-Agent", "TGPT/2.11.0")

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute search request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("search API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResp GoogleSearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse search response: %v", err)
	}

	// Convert to our format
	var results []SearchResult
	for _, item := range searchResp.Items {
		// Validate URL format
		if _, err := url.ParseRequestURI(item.Link); err != nil {
			if verbose {
				fmt.Printf("Warning: Skipping invalid URL: %s\n", item.Link)
			}
			continue
		}
		results = append(results, SearchResult{
			Title:   item.Title,
			URL:     item.Link,
			Snippet: item.Snippet,
		})
	}

	return results, nil
}

// extractContent extracts the main content from a web page using is-fast
func extractContent(pageURL string) (string, error) {
	// Check if is-fast binary exists
	if _, err := exec.LookPath("is-fast"); err != nil {
		return "", fmt.Errorf("is-fast binary not found in PATH. Please install it from https://github.com/Magic-JD/is-fast")
	}

	// Convert Reddit URLs to old.reddit.com for better parsing
	pageURL = strings.Replace(pageURL, "www.reddit.com", "old.reddit.com", 1)

	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), extractionTimeout)
	defer cancel()

	// Use is-fast to extract content
	cmd := exec.CommandContext(ctx, "is-fast", "--direct", pageURL, "--piped")

	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("content extraction timed out for %s", pageURL)
		}
		return "", fmt.Errorf("is-fast extraction failed: %v", err)
	}

	content := strings.TrimSpace(string(output))

	// Limit content length for AI processing
	if len(content) > maxContentPerURL {
		content = content[:maxContentPerURL] + "..."
	}

	return content, nil
}

// formatResultsForAI formats the search results for AI synthesis
func formatResultsForAI(results []SearchResult, originalQuery string) string {
	var formatted strings.Builder

	formatted.WriteString(fmt.Sprintf("Search results for: %s\n\n", originalQuery))

	for i, result := range results {
		formatted.WriteString(fmt.Sprintf("Result %d:\n", i+1))
		formatted.WriteString(fmt.Sprintf("Title: %s\n", result.Title))
		formatted.WriteString(fmt.Sprintf("URL: %s\n", result.URL))
		formatted.WriteString(fmt.Sprintf("Snippet: %s\n", result.Snippet))

		if result.Content != "" {
			formatted.WriteString(fmt.Sprintf("Content: %s\n", result.Content))
		}

		formatted.WriteString("\n---\n\n")
	}

	formatted.WriteString("Please synthesize this information to provide a comprehensive answer to the user's query. Please format response in markdown.")

	result := formatted.String()

	// Limit total length to avoid input limits
	if len(result) > maxTotalContent {
		result = result[:maxTotalContent] + "\n\n[Content truncated due to length...]"
	}

	return result
}

// TestExtractContent is a public wrapper for testing content extraction
func TestExtractContent(url string) (string, error) {
	return extractContent(url)
}

// ExtractSearchParams uses AI to extract optimized search parameters from user input
func ExtractSearchParams(userInput string, aiParams structs.Params, verbose bool) (SearchParams, error) {
	if verbose {
		fmt.Printf("DEBUG: Attempting LLM-based query optimization\n")
	}

	return extractWithRetry(userInput, aiParams, verbose, 2)
}

// extractWithRetry attempts to extract search parameters with retry logic
func extractWithRetry(userInput string, aiParams structs.Params, verbose bool, maxAttempts int) (SearchParams, error) {
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if params, err := attemptLLMExtraction(userInput, aiParams, verbose, attempt); err == nil {
			return params, nil
		}

		if attempt < maxAttempts && verbose {
			fmt.Printf("DEBUG: → Retrying with enhanced prompt\n")
		}
	}

	// All strategies failed, use simple fallback
	if verbose {
		fmt.Printf("DEBUG: ✗ All LLM parsing strategies failed, using simple optimization\n")
	}
	return fallbackToSimple(userInput, verbose), nil
}

// attemptLLMExtraction performs a single attempt to extract search parameters via LLM
func attemptLLMExtraction(userInput string, aiParams structs.Params, verbose bool, attempt int) (SearchParams, error) {
	// Build prompt with structured delimiters
	prompt := buildOptimizationPrompt(userInput, attempt)

	if verbose {
		fmt.Printf("DEBUG: LLM Prompt (attempt %d):\n", attempt)
		fmt.Printf("---START PROMPT---\n%s\n---END PROMPT---\n", prompt)
	}

	response, err := callLLMForOptimization(prompt, aiParams)
	if err != nil {
		if verbose {
			fmt.Printf("DEBUG: LLM call failed on attempt %d (%v)\n", attempt, err)
		}
		return SearchParams{}, err
	}

	if verbose {
		fmt.Printf("DEBUG: LLM Response (attempt %d):\n", attempt)
		fmt.Printf("---START RESPONSE---\n%s\n---END RESPONSE---\n", response)
	}

	// Try multiple parsing strategies
	return parseResponseWithStrategies(response, verbose, attempt)
}

// parseResponseWithStrategies tries multiple parsing strategies in order
func parseResponseWithStrategies(response string, verbose bool, attempt int) (SearchParams, error) {
	// Strategy 1: Look for structured delimiters
	if params, err := parseStructuredResponse(response, verbose); err == nil {
		if verbose {
			fmt.Printf("DEBUG: ✓ Parsed via structured delimiters on attempt %d\n", attempt)
		}
		return validateAndNormalizeParams(params), nil
	}

	// Strategy 2: Forgiving field extraction
	if params, err := parseForgivingResponse(response, verbose); err == nil {
		if verbose {
			fmt.Printf("DEBUG: ✓ Parsed via field extraction on attempt %d\n", attempt)
		}
		return validateAndNormalizeParams(params), nil
	}

	// All parsing strategies failed
	if verbose {
		fmt.Printf("DEBUG: ✗ All parsing strategies failed on attempt %d\n", attempt)
	}

	return SearchParams{}, fmt.Errorf("failed to parse LLM response")
}

// optimizeQuerySimple provides basic query optimization until full LLM integration
func optimizeQuerySimple(userInput string) string {
	query := strings.TrimSpace(userInput)
	lower := strings.ToLower(query)

	// Add current year for time-sensitive queries
	timeKeywords := []string{"latest", "current", "recent", "new", "today", "now", "2024", "2025"}
	hasTimeKeyword := false
	for _, keyword := range timeKeywords {
		if strings.Contains(lower, keyword) {
			hasTimeKeyword = true
			break
		}
	}

	if !hasTimeKeyword {
		// Add 2024 for queries that might benefit from recent results
		if strings.Contains(lower, "best") || strings.Contains(lower, "review") ||
			strings.Contains(lower, "tutorial") || strings.Contains(lower, "guide") {
			query = query + " 2024"
		}
	}

	return query
}

// ConfirmSearchExecution asks user to confirm the search query or auto-confirms for one-shot mode
// inputReader is an optional function to get user input. If nil, uses default bufio.NewReader(os.Stdin)
func ConfirmSearchExecution(params SearchParams, autoConfirm bool, isQuiet bool, inputReader func() (string, error)) bool {
	if autoConfirm {
		// One-shot mode: show informational message unless quiet
		if !isQuiet {
			fmt.Printf("Executing search query: '%s'", params.Query)
			// Show additional parameters if relevant
			if params.SiteFilter != "" {
				fmt.Printf(" (site:%s)", params.SiteFilter)
			}
			if params.NumResults != 5 {
				fmt.Printf(" (%d results)", params.NumResults)
			}
			fmt.Println()
		}
		return true
	}

	// Interactive mode: show confirmation prompt
	fmt.Printf("Execute search query: '%s'", params.Query)

	// Show additional parameters if relevant
	if params.SiteFilter != "" {
		fmt.Printf(" (site:%s)", params.SiteFilter)
	}
	if params.NumResults != 5 {
		fmt.Printf(" (%d results)", params.NumResults)
	}

	fmt.Print(" [y/n]: ")

	// Read user response using custom inputReader or default
	var response string
	var err error

	if inputReader != nil {
		response, err = inputReader()
	} else {
		reader := bufio.NewReader(os.Stdin)
		response, err = reader.ReadString('\n')
	}

	if err != nil {
		return false
	}

	// Check response
	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

// ProcessSearchWithConfirmation handles the full search flow with optimization and confirmation
// inputReader is an optional function to get user input for confirmation. If nil, uses default bufio.NewReader(os.Stdin)
func ProcessSearchWithConfirmation(userInput string, aiParams structs.Params, verbose bool, skipConfirmation bool, isQuiet bool, inputReader func() (string, error)) (string, error) {
	if verbose {
		fmt.Printf("DEBUG: Starting search optimization for: '%s'\n", userInput)
	}

	// Extract optimized search parameters
	searchParams, err := ExtractSearchParams(userInput, aiParams, verbose)
	if err != nil {
		return "", fmt.Errorf("failed to optimize search query: %v", err)
	}

	if verbose {
		fmt.Printf("DEBUG: Optimized query: '%s', results: %d, site: '%s'\n",
			searchParams.Query, searchParams.NumResults, searchParams.SiteFilter)
	}

	// Ask for user confirmation (or auto-confirm for one-shot mode)
	if !ConfirmSearchExecution(searchParams, skipConfirmation, isQuiet, inputReader) {
		return "Search cancelled by user.", nil
	}

	// Proceed with search using the optimized parameters
	return PerformSearchWithParams(searchParams, verbose)
}

// PerformSearchWithParams executes search with pre-built SearchParams
func PerformSearchWithParams(params SearchParams, verbose bool) (string, error) {
	// Get API credentials from environment
	apiKey := os.Getenv("TGPT_GOOGLE_API_KEY")
	searchEngineID := os.Getenv("TGPT_GOOGLE_SEARCH_ENGINE_ID")

	if apiKey == "" || searchEngineID == "" {
		return "", fmt.Errorf("missing required environment variables: TGPT_GOOGLE_API_KEY and TGPT_GOOGLE_SEARCH_ENGINE_ID must be set. Please check SEARCH_SETUP.md for configuration instructions")
	}

	// Perform Google search
	results, err := googleSearch(params, apiKey, searchEngineID, verbose)
	if err != nil {
		return "", fmt.Errorf("search failed: %v", err)
	}

	// Extract content from each result concurrently
	type contentResult struct {
		index   int
		content string
		err     error
	}

	// Limit concurrent extractions to prevent overwhelming the system
	sem := make(chan struct{}, maxConcurrentExtractions)
	var wg sync.WaitGroup
	resultChan := make(chan contentResult)

	var failedCount int64

	for i := range results {
		wg.Add(1)
		go func(idx int, url string) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			if verbose {
				fmt.Printf("Extracting content from result %d: %s\n", idx+1, url)
			}

			content, err := extractContent(url)
			resultChan <- contentResult{index: idx, content: content, err: err}
		}(i, results[i].URL)
	}

	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	for result := range resultChan {
		if result.err != nil {
			atomic.AddInt64(&failedCount, 1)
			// Always provide user feedback for failures
			fmt.Fprintf(os.Stderr, "Warning: Failed to extract content from result %d\n", result.index+1)
			if verbose {
				fmt.Fprintf(os.Stderr, "Details: %v\n", result.err)
			}
			continue
		}
		if verbose {
			fmt.Printf("Successfully extracted %d characters from result %d\n", len(result.content), result.index+1)
		}
		results[result.index].Content = result.content
	}

	// Report summary if there were failures
	if finalFailedCount := atomic.LoadInt64(&failedCount); finalFailedCount > 0 {
		fmt.Fprintf(os.Stderr, "Note: Failed to extract content from %d out of %d results\n", finalFailedCount, len(results))
	}

	// Format results for AI synthesis
	return formatResultsForAI(results, params.Query), nil
}

// callLLMForOptimization calls the LLM to optimize search parameters
func callLLMForOptimization(prompt string, aiParams structs.Params) (string, error) {
	// Use the existing provider system to call LLM
	extraOptions := structs.ExtraOptions{
		IsGetWhole:  true,
		IsGetSilent: true,
	}

	resp, err := providers.NewRequest(prompt, aiParams, extraOptions)
	if err != nil {
		return "", fmt.Errorf("failed to call LLM: %v", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body) // Drain body before closing
		resp.Body.Close()
	}()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body) // Read body for error details
		return "", fmt.Errorf("LLM API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Process the response body
	scanner := bufio.NewScanner(resp.Body)
	fullText := ""

	for scanner.Scan() {
		mainText := providers.GetMainText(scanner.Text(), aiParams.Provider, prompt)
		if len(mainText) < 1 {
			continue
		}
		fullText += mainText
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading LLM response: %v", err)
	}

	// Return the full response - parsing will be handled by the new robust functions
	return strings.TrimSpace(fullText), nil
}

// applySimpleFilters applies basic pattern-based filters to search parameters
func applySimpleFilters(params SearchParams, userInput string) SearchParams {
	lower := strings.ToLower(userInput)

	// Site filter detection
	if strings.Contains(lower, "reddit") {
		params.SiteFilter = "reddit.com"
		params.NumResults = 8 // More results for Reddit discussions
	} else if strings.Contains(lower, "stackoverflow") || strings.Contains(lower, "stack overflow") {
		params.SiteFilter = "stackoverflow.com"
		params.NumResults = 5
	} else if strings.Contains(lower, "github") {
		params.SiteFilter = "github.com"
		params.NumResults = 5
	} else if strings.Contains(lower, "news") || strings.Contains(lower, "latest") {
		params.NumResults = 8 // More results for news/latest info
	}

	return params
}

// buildOptimizationPrompt creates the LLM prompt with structured delimiters
func buildOptimizationPrompt(userInput string, attempt int) string {
	basePrompt := `You are a search query optimizer.

CRITICAL: You must respond with EXACTLY this format:
SEARCH_JSON_START
{"query": "your optimized query", "num_results": 5, "site_filter": ""}
SEARCH_JSON_END

Guidelines:
- Make search terms more specific and effective
- Infer missing context (location, time, etc.) when reasonable
- Suggest appropriate number of results (3-10)
- Only add site_filter if specifically mentioned or highly relevant
- Keep query concise but comprehensive

User request: %s`

	if attempt > 1 {
		basePrompt += `

IMPORTANT: Previous response was not parseable. Follow the EXACT format above with SEARCH_JSON_START and SEARCH_JSON_END delimiters.`
	}

	return fmt.Sprintf(basePrompt, userInput)
}

// parseStructuredResponse extracts JSON from structured delimiters
func parseStructuredResponse(response string, verbose bool) (SearchParams, error) {
	start := strings.Index(response, "SEARCH_JSON_START")
	end := strings.Index(response, "SEARCH_JSON_END")

	if start == -1 || end == -1 {
		if verbose {
			fmt.Printf("DEBUG: Structured delimiters not found in response\n")
		}
		return SearchParams{}, fmt.Errorf("structured delimiters not found")
	}

	jsonText := strings.TrimSpace(response[start+len("SEARCH_JSON_START") : end])

	var params SearchParams
	if err := json.Unmarshal([]byte(jsonText), &params); err != nil {
		if verbose {
			fmt.Printf("DEBUG: JSON parsing failed for structured response: %v\n", err)
		}
		return SearchParams{}, fmt.Errorf("JSON parsing failed: %v", err)
	}

	return params, nil
}

// parseForgivingResponse uses regex to extract fields even from malformed JSON
func parseForgivingResponse(response string, verbose bool) (SearchParams, error) {
	queryRe := regexp.MustCompile(`"query"\s*:\s*"([^"]*)"`)
	numRe := regexp.MustCompile(`"num_results"\s*:\s*(\d+)`)
	siteRe := regexp.MustCompile(`"site_filter"\s*:\s*"([^"]*)"`)

	params := SearchParams{NumResults: 5}

	if match := queryRe.FindStringSubmatch(response); len(match) > 1 {
		params.Query = match[1]
	} else {
		if verbose {
			fmt.Printf("DEBUG: Could not extract query from response\n")
		}
		return SearchParams{}, fmt.Errorf("could not extract query")
	}

	if match := numRe.FindStringSubmatch(response); len(match) > 1 {
		if num, err := strconv.Atoi(match[1]); err == nil {
			params.NumResults = num
		}
	}

	if match := siteRe.FindStringSubmatch(response); len(match) > 1 {
		params.SiteFilter = match[1]
	}

	if verbose {
		fmt.Printf("DEBUG: Extracted fields: query='%s', num_results=%d, site_filter='%s'\n",
			params.Query, params.NumResults, params.SiteFilter)
	}

	return params, nil
}

// validateAndNormalizeParams ensures params are within valid ranges
func validateAndNormalizeParams(params SearchParams) SearchParams {
	// Validate and set defaults
	if params.NumResults < 3 {
		params.NumResults = 3
	}
	if params.NumResults > 10 {
		params.NumResults = 10
	}

	return params
}

// fallbackToSimple provides simple query optimization when LLM fails
func fallbackToSimple(userInput string, verbose bool) SearchParams {
	if verbose {
		fmt.Printf("DEBUG: Using simple query optimization fallback\n")
	}

	params := SearchParams{
		Query:      optimizeQuerySimple(userInput),
		NumResults: 5,
	}
	return applySimpleFilters(params, userInput)
}
