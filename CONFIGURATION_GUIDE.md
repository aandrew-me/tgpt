# TGPT Configuration Guide

*Say goodbye to complex aliases and environment variable conflicts!*

## Quick Start

### 1. Initialize Your Configuration

Create a personalized configuration file in seconds:

```bash
tgpt config init
```

This creates `~/.config/tgpt/config.toml` with an interactive setup that asks for your preferred provider and settings.

### 2. Use TGPT with Your Defaults

```bash
# Before: Complex alias needed
# cai='tgpt --provider openai --key "$CEREBRAS_API_KEY" --model "$CEREBRAS_MODEL" --url "${CEREBRAS_BASE_URL}/chat/completions"'

# After: Simple usage with your configured defaults
tgpt "What is machine learning?"
```

### 3. Quick Profile Switching

```bash
# Use different profiles for different needs
tgpt --profile quick "List files in current directory"
tgpt --profile detailed "Explain quantum computing"
tgpt --profile coding "Write a Python function to sort a list"
```

## Configuration File Structure

Your configuration lives at `~/.config/tgpt/config.toml`. Here's what each section does:

### Default Settings

```toml
[defaults]
provider = "cerebras"          # Your go-to provider
temperature = 0.7              # Response creativity (0.0-2.0)
top_p = 0.9                   # Response diversity (0.0-1.0)
quiet = false                 # Skip loading animations
verbose = false               # Show detailed output
markdown_output = false       # Format output as markdown
search_provider = "is-fast"   # Default search provider
```

### Provider Configurations

Set up all your AI providers once, use them anywhere:

```toml
[providers.cerebras]
type = "openai"               # Uses OpenAI-compatible API
api_key = "${CEREBRAS_API_KEY}"  # Environment variable expansion
model = "qwen-3-coder-480b"
url = "https://api.cerebras.ai/v1/chat/completions"
is_default = true             # Mark as your default

[providers.openai]
type = "openai"
api_key = "${OPENAI_API_KEY}"
model = "gpt-4"
url = ""                      # Uses default OpenAI URL

[providers.gemini]
type = "gemini"
api_key = "${GEMINI_API_KEY}"
model = "gemini-pro"

[providers.deepseek]
type = "deepseek"
api_key = "${DEEPSEEK_API_KEY}"
model = "deepseek-reasoner"
```

### Image Generation Settings

```toml
[image]
default_provider = "pollinations"
width = 1024
height = 1024
ratio = "1:1"
count = "1"
negative_prompt = ""
```

### Search Configuration

```toml
[search]
google_api_key = "${TGPT_GOOGLE_API_KEY}"
google_search_engine_id = "${TGPT_GOOGLE_SEARCH_ENGINE_ID}"
default_provider = "is-fast"
```

### Mode-Specific Settings

Customize behavior for different modes:

```toml
[modes.shell]
auto_execute = false          # Auto-execute with -y flag
preprompt = "You are a helpful shell assistant. Provide concise, accurate commands."

[modes.code]
preprompt = "Generate clean, well-commented, production-ready code with proper error handling."

[modes.interactive]
history_size = 1000
save_conversation = true
```

## Profile System

Profiles let you instantly switch between different configurations for different use cases.

### Pre-configured Profiles

```toml
[profiles.quick]
provider = "cerebras"
quiet = true
temperature = 0.3

[profiles.detailed]
provider = "openai"
verbose = true
temperature = 0.7
markdown_output = true

[profiles.coding]
provider = "cerebras"
temperature = 0.2
```

### Using Profiles

```bash
# Use a profile
tgpt --profile quick "What's the weather?"

# List available profiles
tgpt config profiles

# Create a new profile (via config file editing)
tgpt config edit
```

## Environment Variable Expansion

Use `${VARIABLE_NAME}` syntax to reference environment variables in your config:

```toml
[providers.custom]
api_key = "${MY_API_KEY}"
url = "${BASE_URL}/chat/completions"
```

This keeps sensitive data in environment variables while centralizing other settings.

## Configuration Precedence

TGPT follows a clear precedence order (highest to lowest priority):

1. **Command-line flags** - `tgpt --provider openai "question"`
2. **Environment variables** - `AI_PROVIDER=gemini tgpt "question"`
3. **Profile settings** - `tgpt --profile detailed "question"`
4. **Configuration file** - Your `~/.config/tgpt/config.toml` settings
5. **Built-in defaults** - TGPT's fallback values

### Examples

```bash
# CLI flag overrides everything
tgpt --provider gemini --temperature 0.9 "question"

# Profile used if no CLI flags
tgpt --profile coding "write a function"

# Config file used if no profile or CLI flags
tgpt "default question"
```

## Configuration Commands

### Basic Operations

```bash
# Initialize configuration interactively
tgpt config init

# View current configuration
tgpt config show

# Edit configuration file
tgpt config edit

# Validate your configuration
tgpt config validate
```

### Advanced Operations

```bash
# List configured providers
tgpt config providers

# List available profiles  
tgpt config profiles

# Set a configuration value
tgpt config set defaults.provider cerebras
tgpt config set providers.openai.model gpt-4-turbo

# Get a configuration value
tgpt config get defaults.temperature
```

## Migration from Environment Variables

If you're currently using environment variables, easily migrate to the new configuration system:

### 1. Check What Can Be Migrated

```bash
tgpt config migrate
```

This shows all your current environment variables that can be moved to config.

### 2. Example Migration Output

```
Environment Variables Available for Migration:
========================================

Provider Selection:
  AI_PROVIDER = cerebras
    → Default provider for chat
    Config path: defaults.provider

API Keys:
  CEREBRAS_API_KEY = sk-xxxx
    → Cerebras API key  
    Config path: providers.cerebras.api_key

Model Configuration:
  CEREBRAS_MODEL = qwen-3-coder-480b
    → Cerebras model name
    Config path: providers.cerebras.model
```

### 3. Benefits After Migration

- **Centralized Management**: One config file instead of scattered env vars
- **No More Conflicts**: Isolated from other programs' environment variables
- **Profile Support**: Different configs for different scenarios
- **Better Documentation**: Self-documenting TOML format
- **Environment Expansion**: Still use env vars where needed with `${VAR}` syntax

## Common Usage Patterns

### Daily Developer Workflow

```bash
# Morning standup - quick responses
tgpt --profile quick "What should I focus on today?"

# Code review - detailed analysis
tgpt --profile detailed "Review this pull request"

# Coding session - focused on code generation
tgpt --profile coding "Write unit tests for this function"
```

### Team Collaboration

Share configuration templates:

```bash
# Export your config (remove sensitive keys first)
tgpt config show > team-config-template.toml

# Team members can adapt the template
cp team-config-template.toml ~/.config/tgpt/config.toml
```

### Multi-Provider Setup

Perfect for users with multiple API keys:

```bash
# Different providers for different tasks
tgpt --provider cerebras "Quick coding question"    # Fast, cheap
tgpt --provider openai "Complex analysis"          # High quality
tgpt --provider gemini "Creative writing"          # Different strengths
```

## Troubleshooting

### Configuration File Not Found

```bash
# Create the configuration file
tgpt config init

# Or check the expected location
echo "Config should be at: ~/.config/tgpt/config.toml"
```

### Profile Not Working

```bash
# Check if profile exists
tgpt config profiles

# Validate configuration syntax
tgpt config validate
```

### Environment Variables Not Expanding

```bash
# Check if variables are set
echo $CEREBRAS_API_KEY

# Validate expansion in config
tgpt config validate
```

### Provider Not Found

```bash
# List configured providers
tgpt config providers

# Check the provider name in your config file
tgpt config show
```

### Boolean Flags Ignored

This is expected behavior! Boolean flags from CLI always take precedence:

```bash
# This will be quiet regardless of config file setting
tgpt -q "question"

# This will use config file setting for quiet
tgpt "question"
```

### Temperature/Top-p Not Applied

Check the value format in your config:

```toml
# Correct format
temperature = 0.7

# Incorrect format  
temperature = "0.7"  # Remove quotes for numbers
```

## Configuration Examples

### Minimal Setup for Cerebras Users

```toml
[defaults]
provider = "cerebras"

[providers.cerebras]
type = "openai"
api_key = "${CEREBRAS_API_KEY}"
model = "qwen-3-coder-480b"
url = "https://api.cerebras.ai/v1/chat/completions"
```

### Power User Setup

```toml
[defaults]
provider = "cerebras"
temperature = 0.7
quiet = false

[providers.cerebras]
type = "openai"
api_key = "${CEREBRAS_API_KEY}"
model = "qwen-3-coder-480b"
url = "https://api.cerebras.ai/v1/chat/completions"
is_default = true

[providers.openai]
type = "openai"
api_key = "${OPENAI_API_KEY}"
model = "gpt-4"

[providers.claude]
type = "openai"
api_key = "${ANTHROPIC_API_KEY}"
model = "claude-3-sonnet"
url = "https://api.anthropic.com/v1/messages"

[profiles.work]
provider = "cerebras"
quiet = true
temperature = 0.3

[profiles.research]
provider = "openai"
verbose = true
temperature = 0.8

[profiles.creative]
provider = "claude"
temperature = 1.0
```

### Team/Enterprise Setup

```toml
[defaults]
provider = "azure-openai"
temperature = 0.7
verbose = true

[providers.azure-openai]
type = "openai"
api_key = "${AZURE_OPENAI_API_KEY}"
model = "gpt-4"
url = "${AZURE_OPENAI_ENDPOINT}/openai/deployments/gpt-4/chat/completions"

[providers.internal-llm]
type = "openai"
api_key = "${INTERNAL_API_KEY}"
model = "company-model-v1"
url = "https://internal-ai.company.com/v1/chat/completions"

[modes.code]
preprompt = "Follow company coding standards. Include proper error handling and logging."
```

## Interactive Modes & Usage Patterns

TGPT supports multiple interactive modes, each designed for specific workflows. Configuration applies consistently across all modes with the same precedence rules.

### Mode Overview

| Mode | Flag | Description | Configuration Impact |
|------|------|-------------|---------------------|
| **One-shot** | (default) | Single query and exit | All config applied normally |
| **Interactive** | `-i` | Normal conversation mode | Config + conversation history |
| **Multiline** | `-m` | Multi-line input with TUI | Config + visual interface |
| **Interactive Shell** | `-is` | Shell command assistant | Config + system prompts |
| **Find/Search** | `-f` | One-shot web search | Config + search providers |
| **Interactive Find** | `-if` | Interactive search session | Config + search + history |
| **Interactive Alias** | `-ia` | Shell mode with aliases | Config + alias support |

### Interactive Mode (`-i`)

Normal conversation mode with persistent context and history:

```bash
# Start interactive mode with default config
tgpt -i

# Start with a specific provider and profile
tgpt --profile detailed -i

# Start with initial prompt
tgpt -i "Help me debug this Python code"
```

**Configuration behavior:**
- All config file settings apply
- Profile settings override defaults
- CLI flags override everything
- Conversation history is maintained within session
- Preprompt only applied to first message

**Example configuration for interactive mode:**
```toml
[defaults]
provider = "cerebras"
temperature = 0.7

[modes.interactive]
history_size = 1000
save_conversation = true
preprompt = "You are a helpful coding assistant."

[profiles.chat]
temperature = 0.8
verbose = true
```

### Multiline Mode (`-m`)

TUI-based mode for complex, multi-line inputs:

```bash
# Start multiline mode
tgpt -m

# With quiet mode for cleaner interface
tgpt -m --quiet

# With specific profile
tgpt --profile creative -m
```

**Key features:**
- Press Ctrl+D to submit
- Ctrl+C to exit
- Esc to unfocus, 'i' to focus
- 'p' to paste, 'c' to copy response
- 'b' to copy last code block

**Configuration behavior:**
- Same as interactive mode
- UI respects `quiet` setting
- TUI elements adapt to terminal size

### Interactive Shell Mode (`-is`)

AI assistant that can execute shell commands:

```bash
# Start interactive shell mode
tgpt -is

# With auto-execution enabled
tgpt -is -y

# With specific shell preprompt
tgpt --preprompt "You are a Linux expert" -is
```

**Features:**
- AI wraps commands in `<cmd>` tags
- Confirmation prompt before execution
- Command output added to conversation context
- Shell environment detected automatically

**Configuration:**
```toml
[modes.shell]
auto_execute = false          # Auto-execute with -y flag
preprompt = "You are a helpful shell assistant. Provide concise, accurate commands."

[profiles.sysadmin]
provider = "cerebras"
temperature = 0.2
```

### Interactive Alias Mode (`-ia`)

Enhanced shell mode with access to aliases and functions:

```bash
# Start with alias support
tgpt -ia

# Auto-execute commands
tgpt -ia -y
```

**Differences from regular shell mode:**
- Commands executed with full alias/function support
- Reads shell configuration (`.bashrc`, `.zshrc`, etc.)
- Access to custom shell functions
- Better integration with user's shell environment

### Configuration Precedence in Interactive Modes

All interactive modes follow the same configuration precedence rules, but with some mode-specific considerations:

#### Standard Precedence (applies to all modes)
1. **CLI flags** (highest) - Override everything for that session
2. **Environment variables** - `TGPT_*` variables
3. **Profile settings** - Applied if `--profile` specified
4. **Configuration file** - Your `~/.config/tgpt/config.toml`
5. **Built-in defaults** (lowest) - TGPT fallbacks

#### Mode-Specific Behaviors

**Interactive Sessions (`-i`, `-is`, `-ia`, `-if`):**
- Configuration applied once at startup
- Settings persist for entire session
- Cannot change provider/model mid-session
- CLI flags affect entire interactive session

**Example - Different Approaches:**
```bash
# Approach 1: CLI flags apply to entire session
tgpt --temperature 0.9 -i    # All messages use temp 0.9

# Approach 2: Profile applies to session
tgpt --profile creative -i   # All creative profile settings

# Approach 3: Default config used
tgpt -i                      # Uses config file defaults
```

**One-shot vs Interactive Configuration:**
```bash
# One-shot: Configuration evaluated once
tgpt --temperature 0.3 "Write a function"

# Interactive: Same config for entire session
tgpt --temperature 0.3 -i
> Write a function          # Uses temp 0.3
> Explain the function      # Still uses temp 0.3
> exit

# Different providers per query (one-shot only)
tgpt --provider openai "Complex analysis"
tgpt --provider cerebras "Quick question"

# In interactive mode, provider set for entire session
tgpt --provider openai -i
> Complex analysis          # Uses OpenAI
> Quick question           # Still uses OpenAI (can't switch mid-session)
```

## Search Configuration & Usage

TGPT includes powerful web search capabilities with multiple provider options and AI-enhanced query optimization.

### Search Providers

TGPT supports three search providers:

| Provider | Description | Requirements |
|----------|-------------|--------------|
| **is-fast** | Fast local content extraction | Install `is-fast` binary |
| **firecrawl** | Web scraping service | Firecrawl API account |
| **google** | Google Custom Search | Google API key + Search Engine ID |

### Search Provider Configuration

```toml
[defaults]
search_provider = "is-fast"   # Default search provider

[search]
# Google Custom Search setup
google_api_key = "${TGPT_GOOGLE_API_KEY}"
google_search_engine_id = "${TGPT_GOOGLE_SEARCH_ENGINE_ID}"
default_provider = "is-fast"

# Alternative using environment variables directly
# google_api_key = "${GOOGLE_API_KEY}"
# google_search_engine_id = "${GOOGLE_SEARCH_ENGINE_ID}"
```

### Search Usage Modes

#### One-shot Search (`-f`)

Perform a single search query and exit:

```bash
# Basic search
tgpt -f "What is the latest news about AI?"

# With specific provider (if multiple configured)
tgpt --profile research -f "Python 3.12 new features"

# Verbose mode to see search process
tgpt -f --verbose "Climate change 2024 report"
```

**Configuration behavior:**
- Uses default search provider
- All standard config applies (provider, model, temperature, etc.)
- Search happens automatically without confirmation
- Results processed by AI with configured parameters

#### Interactive Search (`-if`)

Start an interactive session with web search capabilities:

```bash
# Start interactive search mode
tgpt -if

# With specific profile
tgpt --profile research -if
```

**Features:**
- AI determines when web search is needed
- Uses `<search>query</search>` tags to indicate search intent
- Search confirmation shown for each query
- Search results added to conversation context
- Can reference previous search results

**Example interaction:**
```
🔍 ╭─ You
╰─> What's the weather in Paris today?

🤖 ╭─ Bot  
╰─> <search>current weather Paris France today</search>

Search optimized: "current weather Paris France today" (3 results)
Proceed with search? [Y/n]: y

🤖 ╭─ Bot  
╰─> Based on the search results, the current weather in Paris...
```

### Search Provider Setup

#### Is-fast Provider
```bash
# Install is-fast binary
# Visit: https://github.com/aandrew-me/is-fast

# Configuration (default provider)
[defaults]
search_provider = "is-fast"
```

#### Google Custom Search
```bash
# Set environment variables
export TGPT_GOOGLE_API_KEY="your_api_key"
export TGPT_GOOGLE_SEARCH_ENGINE_ID="your_engine_id"
```

```toml
[defaults]
search_provider = "google"

[search]
google_api_key = "${TGPT_GOOGLE_API_KEY}"
google_search_engine_id = "${TGPT_GOOGLE_SEARCH_ENGINE_ID}"
```

#### Firecrawl Provider
```bash
# Set environment variables
export FIRECRAWL_API_KEY="your_firecrawl_key"
```

```toml
[defaults]
search_provider = "firecrawl"

# Additional firecrawl config can be added to [search] section
```

### Search Configuration Precedence

Search functionality follows the same precedence as other features:

1. **CLI flags** - Not available for search provider selection
2. **Environment variables** - `TGPT_SEARCH_PROVIDER`
3. **Profile settings** - `search_provider = "google"`
4. **Config file** - `defaults.search_provider`
5. **Built-in default** - `"is-fast"`

### Search with Profiles

Create specialized profiles for different search scenarios:

```toml
[profiles.research]
provider = "openai"
model = "gpt-4"
temperature = 0.3
search_provider = "google"
verbose = true

[profiles.quick-search]
provider = "cerebras"
temperature = 0.5
search_provider = "is-fast"
quiet = true

[profiles.deep-dive]
provider = "openai"
model = "gpt-4"
temperature = 0.7
search_provider = "google"
```

Usage:
```bash
# Research with high-quality model and Google search
tgpt --profile research -f "academic papers on quantum computing 2024"

# Quick searches with fast provider
tgpt --profile quick-search -f "weather forecast"

# Interactive deep research
tgpt --profile deep-dive -if
```

## Interactive Alias Mode Details

The Interactive Alias Mode (`-ia`) provides enhanced shell integration beyond regular interactive shell mode.

### Key Features

1. **Full Alias Support**: Commands executed with complete shell alias resolution
2. **Function Access**: Shell functions from your `.bashrc`/`.zshrc` are available
3. **Environment Variables**: Full access to your shell environment
4. **Custom Prompts**: Shell-aware AI prompts with environment context

### Configuration

```toml
[modes.interactive_alias]
preprompt = "You are a shell expert with access to all user aliases and functions."
auto_execute = false

[profiles.shell-power]
provider = "cerebras"
temperature = 0.1
```

### Usage Examples

```bash
# Basic usage
tgpt -ia

# With auto-execution
tgpt -ia -y

# With custom profile
tgpt --profile shell-power -ia
```

### Differences from Regular Shell Mode

| Feature | Shell Mode (`-is`) | Alias Mode (`-ia`) |
|---------|-------------------|-------------------|
| Command execution | Basic shell | Full alias/function support |
| Shell config | Not loaded | Loads `.bashrc`/`.zshrc` |
| Environment | Basic | Full user environment |
| Use case | Simple commands | Complex shell workflows |

### Example Interactions

```
╭─ You
╰─> List my docker containers using my custom alias

🤖 ╭─ Bot  
╰─> I'll list your Docker containers using a custom command.
<cmd>dps</cmd>

Execute shell command: `dps` ? [y/n]: y
[Command executes your custom `dps` alias]
```

## Advanced Features

### Environment-Specific Configurations

Use environment variables to switch configs:

```bash
# Development
export TGPT_PROFILE=dev
tgpt "question"  # Uses dev profile automatically

# Production
export TGPT_PROFILE=prod
tgpt "question"  # Uses prod profile automatically
```

### Dynamic URL Construction

```toml
[providers.custom]
api_key = "${API_KEY}"
url = "${BASE_URL}/v${API_VERSION}/chat/completions"
```

```bash
export BASE_URL="https://api.custom.com"
export API_VERSION="1"
export API_KEY="sk-xxxx"
```

## Best Practices

### Security

1. **Never hardcode API keys** - Always use environment variables with `${VAR}` syntax
2. **Keep config files out of version control** - Add `config.toml` to `.gitignore`
3. **Use different API keys for different environments** - Dev, staging, production

### Organization

1. **Use descriptive profile names** - `work`, `personal`, `research` vs `profile1`, `profile2`
2. **Group related providers** - All OpenAI-compatible providers together
3. **Document your setup** - Add comments to config file for team members

### Performance

1. **Set appropriate temperature values** - Lower (0.1-0.3) for factual tasks, higher (0.7-1.0) for creative tasks
2. **Use faster providers for simple tasks** - Save costs and time
3. **Enable quiet mode for scripts** - `quiet = true` in automation profiles

## Migrating from Aliases

### Before (Complex Alias)

```bash
# .bashrc or .zshrc
alias cai='tgpt --provider openai --key "$CEREBRAS_API_KEY" --model "$CEREBRAS_MODEL" --url "${CEREBRAS_BASE_URL}/chat/completions"'
alias gai='tgpt --provider gemini --key "$GEMINI_API_KEY" --model "$GEMINI_MODEL"'
alias quick='tgpt --provider cerebras --quiet --temperature 0.3'
```

### After (Clean Configuration)

```toml
# ~/.config/tgpt/config.toml
[defaults]
provider = "cerebras"

[providers.cerebras]
type = "openai"
api_key = "${CEREBRAS_API_KEY}"
model = "${CEREBRAS_MODEL}"
url = "${CEREBRAS_BASE_URL}/chat/completions"

[providers.gemini]
type = "gemini"
api_key = "${GEMINI_API_KEY}"
model = "${GEMINI_MODEL}"

[profiles.quick]
provider = "cerebras"
quiet = true
temperature = 0.3
```

Usage becomes much cleaner:

```bash
# Simple default usage
tgpt "question"

# Provider switching
tgpt --provider gemini "question"

# Profile usage
tgpt --profile quick "question"
```

## Getting Help

### Built-in Help

```bash
# General help
tgpt --help

# Configuration help
tgpt config --help

# List all config commands
tgpt config
```

### Configuration Validation

```bash
# Check your configuration
tgpt config validate

# View effective configuration
tgpt config show
```

### Community Resources

- **GitHub Issues**: Report bugs and request features
- **Documentation**: Check the main README for updates
- **Examples**: See the `examples/` directory (if available)

---

## Quick Reference Card

| Command | Purpose |
|---------|---------|
| `tgpt config init` | Create configuration file |
| `tgpt config show` | View current configuration |
| `tgpt config edit` | Edit configuration file |
| `tgpt config validate` | Check configuration |
| `tgpt config providers` | List providers |
| `tgpt config profiles` | List profiles |
| `tgpt config migrate` | Migrate from env vars |
| `tgpt --profile NAME` | Use specific profile |
| `tgpt --provider NAME` | Use specific provider |
| `tgpt -i` | Interactive conversation mode |
| `tgpt -m` | Multiline interactive mode |
| `tgpt -is` | Interactive shell mode |
| `tgpt -ia` | Interactive shell with aliases |
| `tgpt -f "query"` | One-shot web search |
| `tgpt -if` | Interactive search mode |

**Configuration File**: `~/.config/tgpt/config.toml`

**Environment Variables**: Use `${VAR_NAME}` syntax

**Precedence**: CLI → Env → Profile → Config → Defaults

---

*With TGPT's new configuration system, you get all the power with none of the complexity. Set it up once, use it everywhere!*