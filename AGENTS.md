# AGENTS.md — tgpt Project Context

## Overview
`tgpt` is a cross-platform CLI tool written in Go (`github.com/aandrew-me/tgpt/v2`) that interacts with 25+ LLM and image-generation providers (both API-key based and free/reverse-engineered endpoints), featuring interactive TUI, web search, local tool execution, and MCP support.

## Key Architecture & Directory Map
- `main.go`: Entry point. Handles CLI flag parsing, interactive loops (using Bubble Tea), piped stdin, tool execution loops, and dispatch to providers or image generators.
- `src/structs/structs.go`: Core data models.
  - `Params`: Model name, API key, provider name, temperature, prompts, message history, tools.
  - `ExtraOptions`: Execution flags (interactive, silent, code-only, command execution, web search).
- `src/providers/`: LLM backend implementations.
  - `providers.go`: Central router (`AvailableProviders`, `NewRequest`). Dispatches requests to individual subpackages (`openai`, `gemini`, `deepseek`, `pollinations`, `opencode`, etc.).
  - Each provider package implements a request function returning `(*http.Response, error)`.
- `src/imagegen/`: Image generation backends (`aihorde`, `anyapi`, `magicstudio`, `pollinations`).
- `src/client/`: HTTP client setup utilizing `bogdanfinn/tls-client` and `bogdanfinn/fhttp` for browser fingerprint emulation and proxy support.
- `src/tools/`: Built-in local tool execution (e.g. bash commands, file operations) and function calling.
- `src/mcp/`: Model Context Protocol client implementation (`mark3labs/mcp-go`).
- `src/search/`: Web search integration (Exa, Google, Serping, etc.) for grounded answers.
- `src/bubbletea/`: Terminal UI components (interactive chat, status bars, spinners).
- `config.conf` / `tgpt.json`: Default configuration and model/provider definitions.

## Build & Test Commands
- Build executable: `go build -o tgpt.exe .`
- Run without compiling: `go run . "your prompt"`
- Run tests: `go test ./...`
- Test specific provider: `go test ./src/providers -run TestProviders`

## Conventions & Rules for Changes
1. **Adding a Provider:**
   - Create package under `src/providers/<name>/<name>.go`.
   - Implement the standard request builder returning `(*http.Response, error)`.
   - Register the provider name in `AvailableProviders` and the switch case in `src/providers/providers.go:NewRequest()`.
2. **HTTP Clients:**
   - Use `client.NewClient()` or `fhttp` instead of raw `net/http` if TLS fingerprinting or custom browser headers are required.
3. **Configuration Precedence:**
   - Command-line flags > Environment variables (`TGPT_*`) > `config.conf` / `tgpt.json`.
