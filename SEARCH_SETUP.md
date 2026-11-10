# Web Search Setup for TGPT

The search functionality provides three modes:
- `-f` (find) - One-shot search with query optimization and confirmation
- `-if` (interactive find) - Interactive search session
- `-ia` (interactive alias) - Interactive shell mode with aliases and functions

This integrates Google Custom Search API with AI-powered content synthesis and query optimization.

## Prerequisites

### 1. is-fast Binary (Required)
The search functionality requires the `is-fast` binary for high-quality web content extraction. Install it from:
[https://github.com/Magic-JD/is-fast](https://github.com/Magic-JD/is-fast)

```bash
# Verify installation
is-fast --help
```

### Why is-fast?
After testing multiple content extraction methods (see `tools/content-extraction-test/README.md`), `is-fast` was chosen because:
- **Best Quality**: Extracts 9,637 words vs alternatives (497-6,436 words)
- **Fast Performance**: 609ms extraction time
- **Superior Formatting**: Maintains proper text structure and readability
- **No API Costs**: Local binary vs paid API services

Alternative methods tested included Colly+Goquery, go-readability, Jina.ai, Tavily, Firecrawl, and ScrapingBee. Full comparison results are available in the testing directory.

## Required Environment Variables

To use the web search feature, you need to set up the following environment variables:

### 1. Google Custom Search API Key
```bash
export TGPT_GOOGLE_API_KEY="your_google_api_key_here"
```

### 2. Google Custom Search Engine ID
```bash
export TGPT_GOOGLE_SEARCH_ENGINE_ID="your_search_engine_id_here"
```

## Setup Instructions

### Step 1: Get Google Custom Search API Key
1. Go to the [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the "Custom Search API"
4. Go to "Credentials" and create an API key
5. Copy the API key

### Step 2: Create Custom Search Engine
1. Go to [Google Custom Search Engine](https://cse.google.com/cse/)
2. Click "Add" to create a new search engine
3. Enter `*` in "Sites to search" to search the entire web
4. Create the search engine
5. Copy the Search Engine ID from the setup page

### Step 3: Set Environment Variables
Add these lines to your shell profile (`.bashrc`, `.zshrc`, etc.):

```bash
export TGPT_GOOGLE_API_KEY="your_actual_api_key"
export TGPT_GOOGLE_SEARCH_ENGINE_ID="your_actual_search_engine_id"
```

Then reload your shell or run `source ~/.bashrc` (or your shell config file).

## Alternative: serpingapi

[serpingapi](https://serpingapi.com) can be used instead of Google Custom Search. It returns Google results and needs a single API key (no search engine ID). A free plan with 1,000 searches per month is available and does not require a card.

1. Create an account at [https://serpingapi.com](https://serpingapi.com) and copy an API key from the dashboard
2. Set the key and select the provider:

```bash
export SERPINGAPI_API_KEY="your_serpingapi_key"
export SEARCH_PROVIDER="serpingapi"   # or pass --search-provider serpingapi
```

Site filters are sent as part of the query (`site:reddit.com ...`). API reference: [https://serpingapi.com/docs](https://serpingapi.com/docs)

## Alternative: Serply

[Serply](https://serply.io) can also be used instead of Google Custom Search. It returns Google results from a single API key with no search engine ID. New accounts start with 2,500 free credits and no card, and purchased credits do not expire.

1. Create an account at [https://serply.io](https://serply.io) and copy an API key from the dashboard
2. Set the key and select the provider:

```bash
export SERPLY_API_KEY="your_serply_key"
export SEARCH_PROVIDER="serply"   # or pass --search-provider serply
```

Site filters are sent as part of the query (`site:reddit.com ...`). API reference: [https://serply.io/docs](https://serply.io/docs)

## Alternative: Keenable (no API key needed)

[Keenable](https://keenable.ai) is a web search index built for AI agents. It works **without an API key** (rate limited per IP) and needs neither a search engine ID nor the `is-fast` binary: each result comes with a long page excerpt, which is passed to the model directly.

```bash
export SEARCH_PROVIDER="keenable"   # or pass --search-provider keenable
tgpt -f "latest Go release notes"
```

An API key is optional and lifts the per-IP limit (100,000 free requests a month):

```bash
export KEENABLE_API_KEY="your_keenable_key"
```

Site filters are sent as Keenable's `site` field. Keenable is also available as a built-in tool (`-t web_search_keenable`). API reference: [https://docs.keenable.ai/api-reference/search](https://docs.keenable.ai/api-reference/search)

## Alternative: Sofya

[Sofya](https://sofya.co) is a web search API for AI agents. It needs a single API key and no `is-fast` binary: searches use `basic` depth, which returns the page text of each result, and that text is passed to the model directly. Each search costs 2 credits. Credits are pay as you go, and GitHub accounts a year or older get free monthly credits.

1. Create an account at [https://sofya.co](https://sofya.co) and copy an API key from the dashboard
2. Set the key and select the provider:

```bash
export SOFYA_API_KEY="your_sofya_key"
export SEARCH_PROVIDER="sofya"   # or pass --search-provider sofya
```

Site filters are sent as Sofya's `include_domains` field. Sofya is also available as a built-in tool (`-t web_search_sofya`). API reference: [https://sofya.co/docs](https://sofya.co/docs)

## Alternative: Parallel

[Parallel](https://docs.parallel.ai/integrations/mcp/search-mcp) is an opt-in
provider using anonymous Search MCP. No API key or `is-fast` binary is required;
source URLs and page excerpts are returned directly to the model. Anonymous
search uses the server-managed Fast default and is subject to free-tier limits.

```bash
tgpt -f --search-provider parallel "Go release notes on go.dev"
# Or select it for find mode through the environment:
export SEARCH_PROVIDER=parallel
# Agent tool mode (requires an inference provider that supports tools):
tgpt -t web_search_parallel "Find the latest Go release notes on go.dev"
```

Site filters use `site:domain` in the search query. The requested result count is
applied locally because anonymous Search MCP does not expose a result-limit
parameter. The built-in `web_search_parallel` tool accepts `query`, optional
`site_filter` (a bare domain), and optional `num_results` (default 5, maximum 20).
It does not register Parallel's other MCP tools or fetch pages separately.

Exa remains the default. `--search-provider` takes precedence over
`SEARCH_PROVIDER`; selecting another provider or excluding this tool from an
explicit `-t` list makes no Parallel requests. As with other built-in tools,
`-t` without a list enables all tools, including Parallel. One-shot `-f` skips
confirmation; interactive `-if` retains it. Parallel errors are surfaced without
switching providers.
The normal find and agent commands still require model inference access.

## Alternative: SpecStream

[SpecStream](https://specstream.ai/search-api) is a web search API that returns a source URL and snippet with each result, and the publish date when the page has one. It needs a single API key and no `is-fast` binary: searches ask for page passages, and that text is passed to the model directly. A search of up to 10 results costs $0.005.

1. Create an API key under Agents & API in your workspace at [https://specstream.ai](https://specstream.ai)
2. Set the key and select the provider:

```bash
export SPECSTREAM_API_KEY="your_specstream_key"
export SEARCH_PROVIDER="specstream"   # or pass --search-provider specstream
```

Site filters are sent as part of the query (`site:reddit.com ...`). SpecStream is also available as a built-in tool (`-t web_search_specstream`). API reference: [https://specstream.ai/search-api](https://specstream.ai/search-api)

## Usage Examples

### One-Shot Search (`-f`)
```bash
# Basic search with query optimization and confirmation
tgpt -f "weather in san francisco"
# Output: Execute search query: 'weather in san francisco 2024' [y/n]: y

# Technical query with automatic optimization  
tgpt -f "best practices go error handling"
# Output: Execute search query: 'best practices go error handling 2024' [y/n]: y

# Site-specific search (automatically detected)
tgpt -f "docker tutorial reddit"
# Output: Execute search query: 'docker tutorial' (site:reddit.com) [y/n]: y

# Verbose mode for debugging
tgpt -f --verbose "latest AI news"
# Shows detailed search process and content extraction
```

### Interactive Search (`-if`)
```bash
# Interactive search session
tgpt -if
# Interactive Find mode started. Press Ctrl + C or type exit to quit.
# You: What's the weather like today?
# Bot: Execute search query: 'current weather forecast [location]' [y/n]: y
```

### Interactive Alias Search (`-ia`)
```bash
# Interactive search session with shell aliases and functions
tgpt -ia
# Interactive Shell mode with aliases started. Press Ctrl + C or type exit to quit.
# You: What files are in my home directory?
# Bot: <search>list files in home directory</search>
# Bot: Execute shell command: `ls ~` ? [y/n]: y
```

## Features

### Search Query Optimization
- **Automatic Query Enhancement**: Optimizes search terms for better results (adds year, improves specificity)
- **Smart Parameter Extraction**: Automatically detects site filters (reddit.com, stackoverflow.com, github.com)
- **Result Count Optimization**: Adjusts number of results based on query type (more for news/discussions)

### User Control & Confirmation
- **Search Confirmation**: Shows optimized query before execution with [y/n] prompt
- **Transparency**: Displays all search parameters (query, site filter, result count)
- **User Cancellation**: Easy to cancel search before execution

### Content Processing
- **High-Quality Extraction**: Uses is-fast for superior content extraction
- **Reddit URL Optimization**: Automatically converts to old.reddit.com for better parsing  
- **AI Synthesis**: Combines search results with markdown formatting
- **Verbose Mode**: Optional detailed debugging with --verbose flag
- **Interactive Alias Mode**: Execute shell commands with access to aliases and functions

## Architecture Flow

1. **User Input** → AI extracts search parameters (query, num_results, site_filter)
2. **Google Search API** → Retrieves search results
3. **Content Extraction** → Fetches and parses content from result URLs using is-fast
4. **AI Synthesis** → Combines search results into comprehensive markdown answer

## Troubleshooting

### "missing required environment variables" error
- Make sure both `TGPT_GOOGLE_API_KEY` and `TGPT_GOOGLE_SEARCH_ENGINE_ID` are set
- For `--search-provider serpingapi`, make sure `SERPINGAPI_API_KEY` is set
- For `--search-provider serply`, make sure `SERPLY_API_KEY` is set
- For `--search-provider sofya`, make sure `SOFYA_API_KEY` is set
- For `--search-provider specstream`, make sure `SPECSTREAM_API_KEY` is set
- Verify the variables are exported: `echo $TGPT_GOOGLE_API_KEY`

### "search API returned status 403" error
- Check if your API key is valid
- Ensure the Custom Search API is enabled in Google Cloud Console
- Verify you haven't exceeded your daily quota

### "failed to extract content" warnings
- This is normal for some websites that block scraping
- The search will continue with other results
- Content extraction uses is-fast for high-quality results

### "is-fast extraction failed" error
- Ensure `is-fast` binary is installed and available in PATH
- Check if the target website is accessible

## Rate Limits

- Google Custom Search API: 100 queries per day (free tier)
- serpingapi: 1,000 searches per month (free plan)
- Serply: 2,500 free credits to start, and credits do not expire
- Sofya: 2 credits per search, pay as you go (free monthly credits for GitHub accounts a year or older)
- SpecStream: $5 per 1,000 searches of up to 10 results
- Consider upgrading to paid tier for higher limits if needed
- The tool defaults to 3 results per query to conserve quota