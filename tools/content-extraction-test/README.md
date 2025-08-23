# Content Extraction Testing Tool

This directory contains tools for comparing different web content extraction methods to evaluate alternatives to the `is-fast` dependency.

## Files

- `test_extraction.go` - Main test program that compares various extraction methods
- `API_SETUP.md` - Instructions for setting up API keys for testing
- `content_quality_analysis.txt` - Analysis results from previous test runs

## Usage

```bash
cd tools/content-extraction-test
go run test_extraction.go
```

The tool will create a `test_outputs/` directory with extracted content and generate a comparison report.

## Supported Methods

- **is-fast** (current implementation)
- **Libraries**: Colly+Goquery, go-readability  
- **APIs**: Jina.ai, Tavily, Firecrawl, ScrapingBee, Scrapfly

## Test Results Summary

From testing on 3 URLs (Wikipedia Go page, Go tutorial, Go blog):

- **is-fast**: 9,637 words, 609ms (best quality)
- **Firecrawl**: 6,436 words, 10.5s (67% content, good for LLMs)
- **Tavily**: 497 words, 6.7s (summaries only)
- **Colly+Goquery**: Poor formatting, text runs together
- **go-readability**: Basic content extraction

**Recommendation**: Keep is-fast for best quality, or use Firecrawl API if switching to API-based solution.