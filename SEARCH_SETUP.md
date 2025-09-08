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
- Consider upgrading to paid tier for higher limits if needed
- The tool defaults to 3 results per query to conserve quota