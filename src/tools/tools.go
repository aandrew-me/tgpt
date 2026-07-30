package tools

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/search"
	http "github.com/bogdanfinn/fhttp"
	"github.com/fatih/color"
)

type FunctionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolSpec struct {
	Type     string       `json:"type"` // "function"
	Function FunctionSpec `json:"function"`
}

type ToolHandler func(ctx context.Context, args map[string]any) (string, error)

type Registry struct {
	mu       sync.RWMutex
	tools    map[string]ToolSpec
	handlers map[string]ToolHandler
}

var DefaultRegistry = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{
		tools:    make(map[string]ToolSpec),
		handlers: make(map[string]ToolHandler),
	}
}

func (r *Registry) RegisterBuiltinTools() {
	r.registerBuiltinTools()
}

type contextKey string

const AutoExecKey contextKey = "auto_exec"

// ConfirmedKey marks that any required user confirmation for the current
// tool call has already been obtained (e.g. by PreConfirm), so the tool
// handler should not prompt again.
const ConfirmedKey contextKey = "confirmed"

var bold = color.New(color.Bold)

// confirmAction prompts the user with a yes/no question and returns true only
// if the user explicitly answers "y" or "yes". A bare Enter (empty input) or
// any other input is treated as "no" so that accidental keystrokes never
// trigger a potentially destructive action.
func confirmAction(prompt string) bool {
	bold.Printf("%s", prompt)
	reader := bufio.NewReader(os.Stdin)
	userIn, _ := reader.ReadString('\n')
	userIn = strings.TrimSpace(strings.ToLower(userIn))
	return userIn == "y" || userIn == "yes"
}

// PreConfirm performs any interactive confirmation required for a tool call
// before execution begins. Callers should invoke this on an undeadlined
// context (i.e. before starting a per-execution timeout) so that the time
// spent waiting on user input does not count against the tool's execution
// budget. It returns (proceed, message): if proceed is false, message
// contains the cancellation message that should be returned as the tool's
// result without running the handler at all.
func PreConfirm(ctx context.Context, name string, argsJSON string) (bool, string) {
	if autoExec, _ := ctx.Value(AutoExecKey).(bool); autoExec {
		return true, ""
	}

	var args map[string]any
	if argsJSON != "" && argsJSON != "{}" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}

	switch name {
	case "execute_command":
		cmdStr, _ := args["command"].(string)
		if !confirmAction(fmt.Sprintf("\nExecute tool shell command: `%s` ? [y/n]: ", cmdStr)) {
			return false, "Command execution cancelled by user."
		}
	case "write_file":
		filePath, _ := args["path"].(string)
		if filePath != "" {
			if _, err := os.Stat(filePath); err == nil {
				if !confirmAction(fmt.Sprintf("\nFile `%s` already exists. Overwrite it? [y/n]: ", filePath)) {
					return false, "File overwrite cancelled by user."
				}
			}
		}
	}

	return true, ""
}

func (r *Registry) Register(spec ToolSpec, handler ToolHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[spec.Function.Name] = spec
	r.handlers[spec.Function.Name] = handler
}

func (r *Registry) Has(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, exists := r.handlers[name]
	return exists
}

func (r *Registry) GetOpenAITools() []any {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]any, 0, len(r.tools))
	for _, t := range r.tools {
		res = append(res, t)
	}
	return res
}

func (r *Registry) ListSpecs() []ToolSpec {
	r.mu.RLock()
	defer r.mu.RUnlock()
	res := make([]ToolSpec, 0, len(r.tools))
	for _, t := range r.tools {
		res = append(res, t)
	}
	return res
}

func (r *Registry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	r.mu.RLock()
	handler, exists := r.handlers[name]
	r.mu.RUnlock()

	if !exists {
		return "", fmt.Errorf("tool %q not found in registry", name)
	}

	var args map[string]any
	if argsJSON != "" && argsJSON != "{}" {
		if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
			return "", fmt.Errorf("failed to parse tool arguments: %w", err)
		}
	} else {
		args = make(map[string]any)
	}

	return handler(ctx, args)
}

func (r *Registry) registerBuiltinTools() {
	// 1. web_search
	r.Register(ToolSpec{
		Type: "function",
		Function: FunctionSpec{
			Name:        "web_search",
			Description: "Search the web for up-to-date information on any topic",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"query": map[string]any{
						"type":        "string",
						"description": "The search query",
					},
				},
				"required": []string{"query"},
			},
		},
	}, func(ctx context.Context, args map[string]any) (string, error) {
		query, _ := args["query"].(string)
		if query == "" {
			return "", fmt.Errorf("query parameter is required")
		}
		params := search.SearchParams{Query: query, NumResults: 5}
		res, err := search.PerformExaMCPSearch(params, false)
		if err != nil {
			return "", fmt.Errorf("search failed: %w", err)
		}
		return res, nil
	})

	// 2. read_directory
	r.Register(ToolSpec{
		Type: "function",
		Function: FunctionSpec{
			Name:        "read_directory",
			Description: "List the contents of a directory",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Directory path (defaults to current directory if empty)",
					},
				},
			},
		},
	}, func(ctx context.Context, args map[string]any) (string, error) {
		dirPath, _ := args["path"].(string)
		if dirPath == "" {
			dirPath = "."
		}
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return "", fmt.Errorf("failed to read directory: %w", err)
		}
		var out string
		for _, entry := range entries {
			kind := "file"
			if entry.IsDir() {
				kind = "dir"
			}
			info, _ := entry.Info()
			size := int64(0)
			if info != nil {
				size = info.Size()
			}
			out += fmt.Sprintf("[%s] %s (%d bytes)\n", kind, entry.Name(), size)
		}
		if out == "" {
			return "Directory is empty.", nil
		}
		return out, nil
	})

	// 3. read_file
	r.Register(ToolSpec{
		Type: "function",
		Function: FunctionSpec{
			Name:        "read_file",
			Description: "Read content from a text file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"path"},
			},
		},
	}, func(ctx context.Context, args map[string]any) (string, error) {
		filePath, _ := args["path"].(string)
		if filePath == "" {
			return "", fmt.Errorf("path parameter is required")
		}
		content, err := os.ReadFile(filePath)
		if err != nil {
			return "", fmt.Errorf("failed to read file: %w", err)
		}
		str := string(content)
		runes := []rune(str)
		if len(runes) > 10000 {
			str = string(runes[:10000]) + "\n... [content truncated]"
		}
		return str, nil
	})

	// 4. execute_command
	r.Register(ToolSpec{
		Type: "function",
		Function: FunctionSpec{
			Name:        "execute_command",
			Description: "Execute a shell command",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"command": map[string]any{
						"type":        "string",
						"description": "The shell command line to execute",
					},
				},
				"required": []string{"command"},
			},
		},
	}, func(ctx context.Context, args map[string]any) (string, error) {
		cmdStr, _ := args["command"].(string)
		if cmdStr == "" {
			return "", fmt.Errorf("command parameter is required")
		}

		autoExec, _ := ctx.Value(AutoExecKey).(bool)
		confirmed, _ := ctx.Value(ConfirmedKey).(bool)
		if !autoExec && !confirmed {
			if !confirmAction(fmt.Sprintf("\nExecute tool shell command: `%s` ? [y/n]: ", cmdStr)) {
				return "Command execution cancelled by user.", nil
			}
		}

		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd.exe", "/C", cmdStr)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", cmdStr)
		}
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Sprintf("Command failed with error: %v\nOutput: %s", err, string(out)), nil
		}
		return string(out), nil
	})

	// 5. web_fetch
	// Fetches content of site, then converts the html to markdown
	r.Register(ToolSpec{
		Type: "function",
		Function: FunctionSpec{
			Name:        "web_fetch",
			Description: "Fetch and clean webpage content into structured Markdown",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"url": map[string]any{
						"type":        "string",
						"description": "The URL to fetch content from",
					},
				},
				"required": []string{"url"},
			},
		},
	}, func(ctx context.Context, args map[string]any) (string, error) {
		fetchURL, _ := args["url"].(string)
		if fetchURL == "" {
			return "", fmt.Errorf("url parameter is required")
		}

		if !strings.HasPrefix(fetchURL, "http://") && !strings.HasPrefix(fetchURL, "https://") {
			fetchURL = "https://" + fetchURL
		}

		httpClient, err := client.NewClient()
		if err != nil {
			return "", fmt.Errorf("failed to create HTTP client: %w", err)
		}

		req, err := http.NewRequestWithContext(ctx, "GET", fetchURL, nil)
		if err != nil {
			return "", fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")

		res, err := httpClient.Do(req)
		if err != nil {
			return "", fmt.Errorf("failed to fetch URL: %w", err)
		}
		defer res.Body.Close()

		if res.StatusCode >= 400 {
			return "", fmt.Errorf("HTTP error %d: %s", res.StatusCode, res.Status)
		}

		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			return "", fmt.Errorf("failed to read response body: %w", err)
		}

		conv := converter.NewConverter(
			converter.WithPlugins(
				base.NewBasePlugin(),
				commonmark.NewCommonmarkPlugin(),
			),
		)

		tagsToRemove := []string{"nav", "footer", "header", "aside", "iframe", "svg"}
		for _, tag := range tagsToRemove {
			conv.Register.TagType(tag, converter.TagTypeRemove, converter.PriorityStandard)
		}

		markdown, err := conv.ConvertString(
			string(bodyBytes),
			converter.WithDomain(fetchURL),
		)
		if err != nil {
			return string(bodyBytes), nil
		}

		maxLen := 12000
		rMarkdown := []rune(markdown)
		if len(rMarkdown) > maxLen {
			markdown = string(rMarkdown[:maxLen]) + "\n\n... [content truncated]"
		}

		return markdown, nil
	})

	// 6. write_file
	r.Register(ToolSpec{
		Type: "function",
		Function: FunctionSpec{
			Name:        "write_file",
			Description: "Write content to a file",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"path": map[string]any{
						"type":        "string",
						"description": "Path to the file to write",
					},
					"content": map[string]any{
						"type":        "string",
						"description": "Content to write to the file",
					},
				},
				"required": []string{"path", "content"},
			},
		},
	}, func(ctx context.Context, args map[string]any) (string, error) {
		filePath, _ := args["path"].(string)
		if filePath == "" {
			return "", fmt.Errorf("path parameter is required")
		}
		content, ok := args["content"].(string)
		if !ok {
			return "", fmt.Errorf("content parameter is required")
		}

		autoExec, _ := ctx.Value(AutoExecKey).(bool)
		confirmed, _ := ctx.Value(ConfirmedKey).(bool)
		if !autoExec && !confirmed {
			if _, err := os.Stat(filePath); err == nil {
				if !confirmAction(fmt.Sprintf("\nFile `%s` already exists. Overwrite it? [y/n]: ", filePath)) {
					return "File overwrite cancelled by user.", nil
				}
			}
		}

		dir := filepath.Dir(filePath)
		if dir != "" && dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return "", fmt.Errorf("failed to create parent directories: %w", err)
			}
		}

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write file: %w", err)
		}

		return fmt.Sprintf("Successfully wrote to %s", filePath), nil
	})
}
