# API Setup Instructions

To test the new API extraction methods, you'll need to set up environment variables for the API keys.

## Required Environment Variables

Set these environment variables before running the test:

```bash
export TAVILY_API_KEY="your_tavily_api_key_here"
export FIRECRAWL_API_KEY="your_firecrawl_api_key_here" 
export SCRAPINGBEE_API_KEY="your_scrapingbee_api_key_here"
export SCRAPFLY_API_KEY="your_scrapfly_api_key_here"
```

## API Service Links

- **Tavily**: https://tavily.com/ - AI-powered search API
- **Firecrawl**: https://firecrawl.dev/ - LLM-optimized web scraping
- **ScrapingBee**: https://scrapingbee.com/ - Web scraping with proxy handling
- **Scrapfly**: https://scrapfly.io/ - Fast web scraping API

## Running the Test

Once you have the API keys set up:

```bash
# Set environment variables
export TAVILY_API_KEY="your_key"
export FIRECRAWL_API_KEY="your_key"
export SCRAPINGBEE_API_KEY="your_key"
export SCRAPFLY_API_KEY="your_key"

# Run the test
go run test_extraction.go
```

## Note

The test program will only run the APIs that have valid keys set. If an API key is missing, that method will show an error in the results.

The old methods (colly-goquery, go-readability, jina-api) are kept in the code but not included in this test run for comparison purposes.