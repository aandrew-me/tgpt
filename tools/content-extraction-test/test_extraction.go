package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gocolly/colly/v2"
	readability "github.com/go-shiori/go-readability"
)

type ExtractionResult struct {
	Method      string
	URL         string
	Content     string
	Duration    time.Duration
	Error       error
	WordCount   int
	CharCount   int
}

type ExtractorFunc func(string) (string, error)

var testURLs = []string{
	"https://en.wikipedia.org/wiki/Go_(programming_language)",
	"https://golang.org/doc/tutorial/getting-started",
	"https://blog.golang.org/declaration-syntax",
}

func main() {
	fmt.Println("Content Extraction Comparison Test")
	fmt.Println("===================================")
	
	// Create output directories for new API methods
	methods := []string{"is-fast", "tavily-api", "firecrawl-api", "scrapingbee-api", "scrapfly-api"}
	for _, method := range methods {
		os.MkdirAll(filepath.Join("test_outputs", method), 0755)
	}
	
	// Initialize extractors (keeping old ones but not using them in this test)
	extractors := map[string]ExtractorFunc{
		"is-fast":          extractWithIsFast,
		"colly-goquery":    extractWithCollyGoquery,     // Not used in this run
		"go-readability":   extractWithGoReadability,    // Not used in this run  
		"jina-api":         extractWithJinaAPI,          // Not used in this run
		"tavily-api":       extractWithTavily,
		"firecrawl-api":    extractWithFirecrawl,
		"scrapingbee-api":  extractWithScrapingBee,
		"scrapfly-api":     extractWithScrapfly,
	}
	
	// Active extractors for this test run
	activeExtractors := map[string]ExtractorFunc{
		"is-fast":          extractors["is-fast"],
		"tavily-api":       extractors["tavily-api"],
		"firecrawl-api":    extractors["firecrawl-api"], 
		"scrapingbee-api":  extractors["scrapingbee-api"],
		"scrapfly-api":     extractors["scrapfly-api"],
	}
	
	var allResults []ExtractionResult
	
	// Test each URL with each extraction method
	for i, url := range testURLs {
		fmt.Printf("\n[%d/%d] Testing URL: %s\n", i+1, len(testURLs), url)
		
		for methodName, extractor := range activeExtractors {
			fmt.Printf("  Testing %s... ", methodName)
			
			start := time.Now()
			content, err := extractor(url)
			duration := time.Since(start)
			
			result := ExtractionResult{
				Method:      methodName,
				URL:         url,
				Content:     content,
				Duration:    duration,
				Error:       err,
				WordCount:   len(strings.Fields(content)),
				CharCount:   len(content),
			}
			
			allResults = append(allResults, result)
			
			if err != nil {
				fmt.Printf("ERROR: %v\n", err)
			} else {
				fmt.Printf("OK (%d words, %v)\n", result.WordCount, duration)
				
				// Save content to file
				filename := fmt.Sprintf("url_%d.txt", i+1)
				filepath := filepath.Join("test_outputs", methodName, filename)
				ioutil.WriteFile(filepath, []byte(content), 0644)
			}
		}
	}
	
	// Generate comparison report
	generateReport(allResults)
}

// Current is-fast implementation
func extractWithIsFast(pageURL string) (string, error) {
	// Check if is-fast binary exists
	if _, err := exec.LookPath("is-fast"); err != nil {
		return "", fmt.Errorf("is-fast binary not found in PATH")
	}

	// Convert Reddit URLs to old.reddit.com for better parsing
	pageURL = strings.Replace(pageURL, "www.reddit.com", "old.reddit.com", 1)

	// Add timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	// Limit content length for comparison
	if len(content) > 50000 {
		content = content[:50000] + "..."
	}

	return content, nil
}

// Colly + Goquery implementation
func extractWithCollyGoquery(pageURL string) (string, error) {
	var content string
	var extractionError error
	
	c := colly.NewCollector(
		colly.UserAgent("tgpt-test/1.0"),
	)
	
	c.SetRequestTimeout(30 * time.Second)
	
	c.OnHTML("html", func(e *colly.HTMLElement) {
		doc := e.DOM
		
		// Remove unwanted elements
		doc.Find("script, style, nav, header, footer, aside, .sidebar, .menu, .navigation, .ad, .advertisement").Remove()
		
		// Try to find main content areas
		var mainContent *goquery.Selection
		
		// Look for semantic HTML5 elements first
		if article := doc.Find("article").First(); article.Length() > 0 {
			mainContent = article
		} else if main := doc.Find("main").First(); main.Length() > 0 {
			mainContent = main
		} else if content := doc.Find("[role='main'], .main-content, .content, .post-content, .article-content").First(); content.Length() > 0 {
			mainContent = content
		} else {
			// Fall back to body but remove more unwanted elements
			mainContent = doc.Find("body")
			mainContent.Find("nav, header, footer, aside, .sidebar, .menu, .comments, .related, .ads").Remove()
		}
		
		// Extract text and clean it up
		text := mainContent.Text()
		text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
		text = strings.TrimSpace(text)
		
		content = text
	})
	
	c.OnError(func(r *colly.Response, err error) {
		extractionError = err
	})
	
	err := c.Visit(pageURL)
	if err != nil {
		return "", err
	}
	
	if extractionError != nil {
		return "", extractionError
	}
	
	// Limit content length
	if len(content) > 50000 {
		content = content[:50000] + "..."
	}
	
	return content, nil
}

// go-readability implementation
func extractWithGoReadability(pageURL string) (string, error) {
	// Fetch the page
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	resp, err := client.Get(pageURL)
	if err != nil {
		return "", fmt.Errorf("failed to fetch page: %v", err)
	}
	defer resp.Body.Close()
	
	// Parse with readability
	article, err := readability.FromReader(resp.Body, nil)
	if err != nil {
		return "", fmt.Errorf("readability parsing failed: %v", err)
	}
	
	// Extract text content from HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
	if err != nil {
		return "", fmt.Errorf("failed to parse readability output: %v", err)
	}
	
	text := doc.Text()
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	
	// Add title if available
	if article.Title != "" {
		text = article.Title + "\n\n" + text
	}
	
	// Limit content length
	if len(text) > 50000 {
		text = text[:50000] + "..."
	}
	
	return text, nil
}

// Jina.ai Reader API implementation
func extractWithJinaAPI(pageURL string) (string, error) {
	client := &http.Client{
		Timeout: 30 * time.Second,
	}
	
	jinaURL := "https://r.jina.ai/" + pageURL
	
	resp, err := client.Get(jinaURL)
	if err != nil {
		return "", fmt.Errorf("jina API request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("jina API returned status %d", resp.StatusCode)
	}
	
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read jina response: %v", err)
	}
	
	text := strings.TrimSpace(string(content))
	
	// Limit content length
	if len(text) > 50000 {
		text = text[:50000] + "..."
	}
	
	return text, nil
}

// Tavily API implementation
func extractWithTavily(pageURL string) (string, error) {
	apiKey := os.Getenv("TAVILY_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("TAVILY_API_KEY environment variable not set")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	
	payload := map[string]interface{}{
		"api_key":       apiKey,
		"query":         pageURL,
		"search_depth":  "basic",
		"include_raw_content": true,
		"max_results":   1,
	}
	
	jsonData, _ := json.Marshal(payload)
	
	resp, err := client.Post("https://api.tavily.com/search", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("tavily API request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("tavily API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Results []struct {
			Content    string `json:"content"`
			RawContent string `json:"raw_content"`
		} `json:"results"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse tavily response: %v", err)
	}
	
	if len(result.Results) == 0 {
		return "", fmt.Errorf("no results from tavily")
	}
	
	content := result.Results[0].Content
	if content == "" && result.Results[0].RawContent != "" {
		content = result.Results[0].RawContent
	}
	
	// Clean up content
	content = strings.TrimSpace(content)
	
	// Limit content length
	if len(content) > 50000 {
		content = content[:50000] + "..."
	}
	
	return content, nil
}

// Firecrawl API implementation
func extractWithFirecrawl(pageURL string) (string, error) {
	apiKey := os.Getenv("FIRECRAWL_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("FIRECRAWL_API_KEY environment variable not set")
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	
	payload := map[string]interface{}{
		"url": pageURL,
		"formats": []string{"markdown"},
		"onlyMainContent": true,
	}
	
	jsonData, _ := json.Marshal(payload)
	
	req, err := http.NewRequest("POST", "https://api.firecrawl.dev/v1/scrape", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}
	
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("firecrawl API request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("firecrawl API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Data struct {
			Markdown string `json:"markdown"`
			Content  string `json:"content"`
		} `json:"data"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to parse firecrawl response: %v", err)
	}
	
	content := result.Data.Markdown
	if content == "" {
		content = result.Data.Content
	}
	
	content = strings.TrimSpace(content)
	
	// Limit content length
	if len(content) > 50000 {
		content = content[:50000] + "..."
	}
	
	return content, nil
}

// ScrapingBee API implementation  
func extractWithScrapingBee(pageURL string) (string, error) {
	apiKey := os.Getenv("SCRAPINGBEE_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("SCRAPINGBEE_API_KEY environment variable not set")
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	
	// Build API URL with parameters
	baseURL := "https://app.scrapingbee.com/api/v1/"
	params := url.Values{}
	params.Set("api_key", apiKey)
	params.Set("url", pageURL)
	params.Set("render_js", "false")  // Basic extraction first
	params.Set("extract_rules", `{"content": "body"}`)
	
	apiURL := baseURL + "?" + params.Encode()
	
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("scrapingbee API request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("scrapingbee API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read scrapingbee response: %v", err)
	}
	
	// Parse HTML and extract text
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(string(content)))
	if err != nil {
		// If HTML parsing fails, return raw content
		text := strings.TrimSpace(string(content))
		if len(text) > 50000 {
			text = text[:50000] + "..."
		}
		return text, nil
	}
	
	// Remove unwanted elements and extract text
	doc.Find("script, style, nav, header, footer, aside").Remove()
	text := doc.Text()
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)
	
	// Limit content length
	if len(text) > 50000 {
		text = text[:50000] + "..."
	}
	
	return text, nil
}

// Scrapfly API implementation
func extractWithScrapfly(pageURL string) (string, error) {
	apiKey := os.Getenv("SCRAPFLY_API_KEY")
	if apiKey == "" {
		return "", fmt.Errorf("SCRAPFLY_API_KEY environment variable not set")
	}
	
	client := &http.Client{Timeout: 30 * time.Second}
	
	// Build API URL with parameters
	baseURL := "https://api.scrapfly.io/scrape"
	params := url.Values{}
	params.Set("key", apiKey)
	params.Set("url", pageURL)
	params.Set("render_js", "false")  // Basic extraction first
	params.Set("format", "text")      // Request text format
	
	apiURL := baseURL + "?" + params.Encode()
	
	resp, err := client.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("scrapfly API request failed: %v", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("scrapfly API returned status %d: %s", resp.StatusCode, string(body))
	}
	
	var result struct {
		Result struct {
			Content string `json:"content"`
		} `json:"result"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		// If JSON parsing fails, try reading as plain text
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read scrapfly response: %v", err)
		}
		content := strings.TrimSpace(string(body))
		if len(content) > 50000 {
			content = content[:50000] + "..."
		}
		return content, nil
	}
	
	content := strings.TrimSpace(result.Result.Content)
	
	// If we got HTML, extract text from it
	if strings.Contains(content, "<html") || strings.Contains(content, "<body") {
		doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
		if err == nil {
			doc.Find("script, style, nav, header, footer, aside").Remove()
			text := doc.Text()
			text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
			content = strings.TrimSpace(text)
		}
	}
	
	// Limit content length
	if len(content) > 50000 {
		content = content[:50000] + "..."
	}
	
	return content, nil
}

func generateReport(results []ExtractionResult) {
	fmt.Println("\n\n=== EXTRACTION COMPARISON REPORT ===")
	
	// Group results by method
	methodStats := make(map[string]struct {
		SuccessCount int
		TotalCount   int
		TotalWords   int
		TotalChars   int
		TotalTime    time.Duration
		Errors       []string
	})
	
	for _, result := range results {
		stats := methodStats[result.Method]
		stats.TotalCount++
		stats.TotalTime += result.Duration
		
		if result.Error != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("%s: %v", result.URL, result.Error))
		} else {
			stats.SuccessCount++
			stats.TotalWords += result.WordCount
			stats.TotalChars += result.CharCount
		}
		
		methodStats[result.Method] = stats
	}
	
	// Print summary statistics
	fmt.Println("\nMethod Comparison:")
	fmt.Println("------------------")
	
	activeMethods := []string{"is-fast", "tavily-api", "firecrawl-api", "scrapingbee-api", "scrapfly-api"}
	for _, method := range activeMethods {
		stats := methodStats[method]
		successRate := float64(stats.SuccessCount) / float64(stats.TotalCount) * 100
		avgWords := 0
		avgTime := time.Duration(0)
		
		if stats.SuccessCount > 0 {
			avgWords = stats.TotalWords / stats.SuccessCount
			avgTime = stats.TotalTime / time.Duration(stats.SuccessCount)
		}
		
		fmt.Printf("%-15s | Success: %d/%d (%.1f%%) | Avg Words: %d | Avg Time: %v\n",
			method, stats.SuccessCount, stats.TotalCount, successRate, avgWords, avgTime)
		
		if len(stats.Errors) > 0 {
			fmt.Printf("                  Errors:\n")
			for _, err := range stats.Errors {
				fmt.Printf("                    - %s\n", err)
			}
		}
	}
	
	// Save detailed report
	reportFile := "test_outputs/comparison_report.txt"
	f, err := os.Create(reportFile)
	if err != nil {
		log.Printf("Failed to create report file: %v", err)
		return
	}
	defer f.Close()
	
	fmt.Fprintf(f, "Content Extraction Comparison Report\n")
	fmt.Fprintf(f, "Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
	
	for _, method := range activeMethods {
		stats := methodStats[method]
		fmt.Fprintf(f, "=== %s ===\n", strings.ToUpper(method))
		fmt.Fprintf(f, "Success Rate: %d/%d\n", stats.SuccessCount, stats.TotalCount)
		fmt.Fprintf(f, "Total Words: %d\n", stats.TotalWords)
		fmt.Fprintf(f, "Total Time: %v\n", stats.TotalTime)
		
		if len(stats.Errors) > 0 {
			fmt.Fprintf(f, "Errors:\n")
			for _, err := range stats.Errors {
				fmt.Fprintf(f, "  - %s\n", err)
			}
		}
		fmt.Fprintf(f, "\n")
	}
	
	fmt.Printf("\nDetailed report saved to: %s\n", reportFile)
	fmt.Println("Individual extractions saved in test_outputs/ directories")
	fmt.Println("\nNext steps:")
	fmt.Println("1. Review the extracted content in test_outputs/")
	fmt.Println("2. Compare quality manually for a few URLs")
	fmt.Println("3. Test with AI to see which provides better context")
}