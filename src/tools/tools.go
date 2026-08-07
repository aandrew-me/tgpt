package tools

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"

	"github.com/JohannesKaufmann/html-to-markdown/v2/converter"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/base"
	"github.com/JohannesKaufmann/html-to-markdown/v2/plugin/commonmark"
	"github.com/aandrew-me/tgpt/v2/src/bubbletea"
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

var AllBuiltinTools = []string{
	"web_search_exa",
	"web_search_firecrawl",
	"read_directory",
	"read_file",
	"execute_command",
	"web_fetch",
	"write_file",
	"edit_file",
	"grep",
	"glob",
}

func IsBuiltinTool(name string) bool {
	name = strings.ToLower(strings.TrimSpace(name))
	for _, t := range AllBuiltinTools {
		if t == name {
			return true
		}
	}
	return false
}

func ParseToolList(input string) ([]string, bool) {
	if input == "" {
		return nil, false
	}
	parts := strings.Split(input, ",")
	var toolsList []string
	for _, p := range parts {
		trimmed := strings.ToLower(strings.TrimSpace(p))
		if trimmed == "" {
			continue
		}
		if trimmed == "all" || IsBuiltinTool(trimmed) {
			toolsList = append(toolsList, trimmed)
		} else {
			return nil, false
		}
	}
	if len(toolsList) == 0 {
		return nil, false
	}
	return toolsList, true
}

func (r *Registry) RegisterBuiltinTools(selectedTools ...string) {
	r.registerBuiltinTools(selectedTools...)
}

type contextKey string

const AutoExecKey contextKey = "auto_exec"

// ConfirmedKey marks that any required user confirmation for the current
// tool call has already been obtained (e.g. by PreConfirm), so the tool
// handler should not prompt again.
const ConfirmedKey contextKey = "confirmed"

var bold = color.New(color.Bold)

// confirmAction prompts the user with an interactive yes/no question using Bubble Tea.
func confirmAction(prompt string) (bool, error) {
	return bubbletea.ConfirmMenu(prompt, true)
}

// PreConfirm performs any interactive confirmation required for a tool call
// before execution begins. Callers should invoke this on an undeadlined
// context (i.e. before starting a per-execution timeout) so that the time
// spent waiting on user input does not count against the tool's execution
// budget. It returns (proceed, message, err): if proceed is false, message
// contains the cancellation message that should be returned as the tool's
// result without running the handler at all.
func PreConfirm(ctx context.Context, name string, argsJSON string) (bool, string, error) {
	if autoExec, _ := ctx.Value(AutoExecKey).(bool); autoExec {
		return true, "", nil
	}

	var args map[string]any
	if argsJSON != "" && argsJSON != "{}" {
		_ = json.Unmarshal([]byte(argsJSON), &args)
	}

	switch name {
	case "execute_command":
		cmdStr, _ := args["command"].(string)
		confirmed, err := confirmAction(fmt.Sprintf("\nExecute tool shell command: `%s` ?", cmdStr))
		if err != nil {
			if errors.Is(err, bubbletea.ErrCanceled) {
				return false, "Command execution cancelled by user.", nil
			}
			return false, "", err
		}
		if !confirmed {
			return false, "Command execution cancelled by user.", nil
		}
	case "write_file":
		filePath, _ := args["path"].(string)
		appendMode, _ := args["append"].(bool)
		if filePath != "" {
			if _, err := os.Stat(filePath); err == nil {
				var prompt string
				var cancelMsg string
				if appendMode {
					prompt = fmt.Sprintf("\nAppend to file `%s` ?", filePath)
					cancelMsg = "File append cancelled by user."
				} else {
					prompt = fmt.Sprintf("\nFile `%s` already exists. Overwrite it?", filePath)
					cancelMsg = "File overwrite cancelled by user."
				}
				confirmed, err := confirmAction(prompt)
				if err != nil {
					if errors.Is(err, bubbletea.ErrCanceled) {
						return false, cancelMsg, nil
					}
					return false, "", err
				}
				if !confirmed {
					return false, cancelMsg, nil
				}
			}
		}
	case "edit_file":
		filePath, _ := args["path"].(string)
		if filePath != "" {
			if _, err := os.Stat(filePath); err == nil {
				confirmed, err := confirmAction(fmt.Sprintf("\nEdit file `%s` ?", filePath))
				if err != nil {
					if errors.Is(err, bubbletea.ErrCanceled) {
						return false, "File edit cancelled by user.", nil
					}
					return false, "", err
				}
				if !confirmed {
					return false, "File edit cancelled by user.", nil
				}
			}
		}
	}

	return true, "", nil
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

func (r *Registry) registerBuiltinTools(selectedTools ...string) {
	for _, t := range selectedTools {
		trimmed := strings.ToLower(strings.TrimSpace(t))
		if trimmed == "" || trimmed == "all" || IsBuiltinTool(trimmed) {
			continue
		}
		fmt.Fprintf(os.Stderr, "Warning: unknown tool %q ignored. Available tools: %s\n", t, strings.Join(AllBuiltinTools, ", "))
	}

	shouldRegister := func(name string) bool {
		if len(selectedTools) == 0 {
			return true
		}
		for _, t := range selectedTools {
			trimmed := strings.ToLower(strings.TrimSpace(t))
			if trimmed == "all" || trimmed == name {
				return true
			}
		}
		return false
	}

	// 1. web_search_exa
	if shouldRegister("web_search_exa") {
		r.Register(ToolSpec{
			Type: "function",
			Function: FunctionSpec{
				Name:        "web_search_exa",
				Description: "Search the web for up-to-date information on any topic using Exa",
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
	}

	// 2. web_search_firecrawl
	if shouldRegister("web_search_firecrawl") {
		r.Register(ToolSpec{
			Type: "function",
			Function: FunctionSpec{
				Name:        "web_search_firecrawl",
				Description: "Search the web for up-to-date information on any topic using Firecrawl",
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
			res, err := search.PerformFirecrawlMCPSearch(params, false)
			if err != nil {
				return "", fmt.Errorf("search failed: %w", err)
			}
			return res, nil
		})
	}

	// 2. read_directory
	if shouldRegister("read_directory") {
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
	}

	// 3. read_file
	if shouldRegister("read_file") {
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
	}

	// 4. execute_command
	if shouldRegister("execute_command") {
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
				c, err := confirmAction(fmt.Sprintf("\nExecute tool shell command: `%s` ?", cmdStr))
				if err != nil {
					if errors.Is(err, bubbletea.ErrCanceled) {
						return "Command execution cancelled by user.", nil
					}
					return "", err
				}
				if !c {
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
	}

	// 5. web_fetch
	// Fetches content of site, then converts the html to markdown
	if shouldRegister("web_fetch") {
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

			httpClient, err := client.NewClient(15)
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
	}

	// 6. write_file
	if shouldRegister("write_file") {
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
						"append": map[string]any{
							"type":        "boolean",
							"description": "If true, append content to the file instead of overwriting it (defaults to false)",
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
			appendMode, _ := args["append"].(bool)

			autoExec, _ := ctx.Value(AutoExecKey).(bool)
			confirmed, _ := ctx.Value(ConfirmedKey).(bool)
			if !autoExec && !confirmed {
				if _, err := os.Stat(filePath); err == nil {
					var prompt string
					var cancelMsg string
					if appendMode {
						prompt = fmt.Sprintf("\nAppend to file `%s` ?", filePath)
						cancelMsg = "File append cancelled by user."
					} else {
						prompt = fmt.Sprintf("\nFile `%s` already exists. Overwrite it?", filePath)
						cancelMsg = "File overwrite cancelled by user."
					}
					c, err := confirmAction(prompt)
					if err != nil {
						if errors.Is(err, bubbletea.ErrCanceled) {
							return cancelMsg, nil
						}
						return "", err
					}
					if !c {
						return cancelMsg, nil
					}
				}
			}

			dir := filepath.Dir(filePath)
			if dir != "" && dir != "." {
				if err := os.MkdirAll(dir, 0755); err != nil {
					return "", fmt.Errorf("failed to create parent directories: %w", err)
				}
			}

			if appendMode {
				f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return "", fmt.Errorf("failed to open file for appending: %w", err)
				}
				defer f.Close()
				if _, err := f.WriteString(content); err != nil {
					return "", fmt.Errorf("failed to append to file: %w", err)
				}
				return fmt.Sprintf("Successfully appended to %s", filePath), nil
			}

			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return "", fmt.Errorf("failed to write file: %w", err)
			}

			return fmt.Sprintf("Successfully wrote to %s", filePath), nil
		})
	}

	// 7. edit_file
	if shouldRegister("edit_file") {
		r.Register(ToolSpec{
			Type: "function",
			Function: FunctionSpec{
				Name:        "edit_file",
				Description: "Edit a file by replacing old_content with new_content",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"path": map[string]any{
							"type":        "string",
							"description": "Path to the file to edit",
						},
						"old_content": map[string]any{
							"type":        "string",
							"description": "Exact text or code block in the file to be replaced",
						},
						"new_content": map[string]any{
							"type":        "string",
							"description": "New text or code block to replace old_content with",
						},
					},
					"required": []string{"path", "old_content", "new_content"},
				},
			},
		}, func(ctx context.Context, args map[string]any) (string, error) {
			filePath, _ := args["path"].(string)
			if filePath == "" {
				return "", fmt.Errorf("path parameter is required")
			}
			oldContent, ok := args["old_content"].(string)
			if !ok {
				return "", fmt.Errorf("old_content parameter is required")
			}
			if oldContent == "" {
				return "", fmt.Errorf("old_content parameter cannot be empty")
			}
			newContent, ok := args["new_content"].(string)
			if !ok {
				return "", fmt.Errorf("new_content parameter is required")
			}
			if oldContent == newContent {
				return "", fmt.Errorf("old_content and new_content are identical; no changes to make")
			}

			if _, err := os.Stat(filePath); err != nil {
				return "", fmt.Errorf("failed to read file for editing: %w", err)
			}

			autoExec, _ := ctx.Value(AutoExecKey).(bool)
			confirmed, _ := ctx.Value(ConfirmedKey).(bool)
			if !autoExec && !confirmed {
				c, err := confirmAction(fmt.Sprintf("\nEdit file `%s` ?", filePath))
				if err != nil {
					if errors.Is(err, bubbletea.ErrCanceled) {
						return "File edit cancelled by user.", nil
					}
					return "", err
				}
				if !c {
					return "File edit cancelled by user.", nil
				}
			}

			data, err := os.ReadFile(filePath)
			if err != nil {
				return "", fmt.Errorf("failed to read file for editing: %w", err)
			}

			fileStr := string(data)
			count := strings.Count(fileStr, oldContent)
			if count == 0 {
				return "", fmt.Errorf("old_content not found in %s", filePath)
			}
			if count > 1 {
				return "", fmt.Errorf("old_content appears %d times in %s; please provide more surrounding context to make it unique", count, filePath)
			}

			updatedStr := strings.Replace(fileStr, oldContent, newContent, 1)
			if err := os.WriteFile(filePath, []byte(updatedStr), 0644); err != nil {
				return "", fmt.Errorf("failed to write edited file: %w", err)
			}

			return fmt.Sprintf("Successfully edited %s", filePath), nil
		})
	}

	// 8. grep
	if shouldRegister("grep") {
		r.Register(ToolSpec{
			Type: "function",
			Function: FunctionSpec{
				Name:        "grep",
				Description: "Search file contents using regular expressions",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"pattern": map[string]any{
							"type":        "string",
							"description": "Regular expression pattern to search for",
						},
						"path": map[string]any{
							"type":        "string",
							"description": "Directory or file path to search in (defaults to current directory)",
						},
						"include": map[string]any{
							"type":        "string",
							"description": "Optional file pattern filter (e.g. '*.go' or 'go')",
						},
					},
					"required": []string{"pattern"},
				},
			},
		}, func(ctx context.Context, args map[string]any) (string, error) {
			pattern, _ := args["pattern"].(string)
			if pattern == "" {
				return "", fmt.Errorf("pattern parameter is required")
			}
			re, err := regexp.Compile(pattern)
			if err != nil {
				return "", fmt.Errorf("invalid regular expression: %w", err)
			}

			searchPath, _ := args["path"].(string)
			if searchPath == "" {
				searchPath = "."
			}
			include, _ := args["include"].(string)
			if include != "" && !strings.Contains(include, "*") && !strings.HasPrefix(include, ".") {
				include = "*." + include
			}

			info, err := os.Stat(searchPath)
			if err != nil {
				return "", fmt.Errorf("failed to access path: %w", err)
			}

			var matches []string
			matchCount := 0
			maxMatches := 200

			searchFile := func(filePath string) error {
				if include != "" {
					matched, err := filepath.Match(include, filepath.Base(filePath))
					if err != nil || !matched {
						return nil
					}
				}

				f, err := os.Open(filePath)
				if err != nil {
					return nil
				}
				defer f.Close()

				head := make([]byte, 512)
				n, err := f.Read(head)
				if err != nil && err != io.EOF {
					return nil
				}
				if bytes.IndexByte(head[:n], 0) != -1 {
					return nil
				}
				if _, err := f.Seek(0, io.SeekStart); err != nil {
					return nil
				}

				scanner := bufio.NewScanner(f)
				buf := make([]byte, 64*1024)
				scanner.Buffer(buf, 1024*1024)

				lineNum := 0
				for scanner.Scan() {
					lineNum++
					line := scanner.Text()
					if re.MatchString(line) {
						matches = append(matches, fmt.Sprintf("%s:%d:%s", filePath, lineNum, line))
						matchCount++
						if matchCount >= maxMatches {
							break
						}
					}
				}
				_ = scanner.Err()
				return nil
			}

			if !info.IsDir() {
				_ = searchFile(searchPath)
			} else {
				err = filepath.WalkDir(searchPath, func(p string, d fs.DirEntry, err error) error {
					if err != nil {
						return nil
					}
					if d.IsDir() {
						name := d.Name()
						if strings.HasPrefix(name, ".") && name != "." && name != ".." {
							return filepath.SkipDir
						}
						if name == "node_modules" || name == "vendor" {
							return filepath.SkipDir
						}
						return nil
					}
					if matchCount >= maxMatches {
						return filepath.SkipAll
					}
					return searchFile(p)
				})
				if err != nil {
					return "", fmt.Errorf("search failed: %w", err)
				}
			}

			if len(matches) == 0 {
				return "No matches found.", nil
			}

			out := strings.Join(matches, "\n")
			if matchCount >= maxMatches {
				out += fmt.Sprintf("\n... [truncated at %d matches]", maxMatches)
			}
			runes := []rune(out)
			if len(runes) > 10000 {
				out = string(runes[:10000]) + "\n... [content truncated]"
			}
			return out, nil
		})
	}

	// 9. glob
	if shouldRegister("glob") {
		r.Register(ToolSpec{
			Type: "function",
			Function: FunctionSpec{
				Name:        "glob",
				Description: "Find files and directories matching a glob pattern",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"pattern": map[string]any{
							"type":        "string",
							"description": "Glob pattern to match files (e.g. '*.go', '**/*.json', 'src/*.go')",
						},
						"path": map[string]any{
							"type":        "string",
							"description": "Directory path to start search from (defaults to current directory)",
						},
					},
					"required": []string{"pattern"},
				},
			},
		}, func(ctx context.Context, args map[string]any) (string, error) {
			pattern, _ := args["pattern"].(string)
			if pattern == "" {
				return "", fmt.Errorf("pattern parameter is required")
			}
			searchPath, _ := args["path"].(string)
			if searchPath == "" {
				searchPath = "."
			}

			var matches []string
			maxMatches := 500

			cleanPattern := filepath.ToSlash(pattern)
			hasPath := strings.Contains(cleanPattern, "/")

			err := filepath.WalkDir(searchPath, func(p string, d fs.DirEntry, err error) error {
				if err != nil {
					return nil
				}
				if d.IsDir() {
					name := d.Name()
					if strings.HasPrefix(name, ".") && name != "." && name != ".." {
						return filepath.SkipDir
					}
					if name == "node_modules" || name == "vendor" {
						return filepath.SkipDir
					}
				}

				relPath, _ := filepath.Rel(searchPath, p)
				slashRel := filepath.ToSlash(relPath)

				var matched bool
				if hasPath {
					if strings.Contains(cleanPattern, "**") {
						parts := strings.Split(cleanPattern, "**")
						for i, part := range parts {
							parts[i] = regexp.QuoteMeta(part)
						}
						regexStr := "^" + strings.Join(parts, ".*") + "$"
						regexStr = strings.ReplaceAll(regexStr, `\*`, `[^/]*`)
						regexStr = strings.ReplaceAll(regexStr, `\?`, `[^/]`)
						if re, err := regexp.Compile(regexStr); err == nil {
							matched = re.MatchString(slashRel)
						}
					} else {
						matched, _ = filepath.Match(cleanPattern, slashRel)
					}
				} else {
					matched, _ = filepath.Match(pattern, d.Name())
				}

				if matched {
					if d.IsDir() {
						matches = append(matches, p+"/")
					} else {
						matches = append(matches, p)
					}
					if len(matches) >= maxMatches {
						return filepath.SkipAll
					}
				}
				return nil
			})
			if err != nil {
				return "", fmt.Errorf("glob search failed: %w", err)
			}

			if len(matches) == 0 {
				return "No matching files found.", nil
			}

			return strings.Join(matches, "\n"), nil
		})
	}
}
