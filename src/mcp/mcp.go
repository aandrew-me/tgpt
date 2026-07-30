package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/aandrew-me/tgpt/v2/src/tools"
	mcpclient "github.com/mark3labs/mcp-go/client"
	"github.com/mark3labs/mcp-go/mcp"
)

type ServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     []string          `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
}

type Config struct {
	MCPServers map[string]ServerConfig `json:"mcpServers"`
}

type Manager struct {
	mu       sync.Mutex
	clients  map[string]mcpclient.MCPClient
	registry *tools.Registry
}

func NewManager(registry *tools.Registry) *Manager {
	if registry == nil {
		registry = tools.DefaultRegistry
	}
	return &Manager{
		clients:  make(map[string]mcpclient.MCPClient),
		registry: registry,
	}
}

func LoadConfig(path string) (*Config, error) {
	if path == "" {
		defaultPaths := []string{
			"mcp_config.json",
		}
		if homeDir, err := os.UserHomeDir(); err == nil {
			defaultPaths = append(defaultPaths, filepath.Join(homeDir, ".config", "tgpt", "mcp_config.json"))
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				path = p
				break
			}
		}
	}

	if path == "" {
		return nil, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse MCP config JSON: %w", err)
	}

	return &cfg, nil
}

func (m *Manager) InitServer(ctx context.Context, name string, sc ServerConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var c mcpclient.MCPClient
	var err error

	if sc.URL != "" {
		c, err = mcpclient.NewSSEMCPClient(sc.URL)
		if err != nil {
			return fmt.Errorf("failed to create SSE MCP client for %s: %w", name, err)
		}
	} else if sc.Command != "" {
		c, err = mcpclient.NewStdioMCPClient(sc.Command, sc.Env, sc.Args...)
		if err != nil {
			return fmt.Errorf("failed to create stdio MCP client for %s: %w", name, err)
		}
	} else {
		return fmt.Errorf("invalid server config for %s: missing command or url", name)
	}

	initReq := mcp.InitializeRequest{}
	initReq.Params.ProtocolVersion = mcp.LATEST_PROTOCOL_VERSION
	initReq.Params.ClientInfo = mcp.Implementation{
		Name:    "tgpt-cli",
		Version: "1.0.0",
	}

	_, err = c.Initialize(ctx, initReq)
	if err != nil {
		c.Close()
		return fmt.Errorf("failed to initialize MCP client for %s: %w", name, err)
	}

	m.clients[name] = c

	// List tools and register them
	listToolsReq := mcp.ListToolsRequest{}
	res, err := c.ListTools(ctx, listToolsReq)
	if err != nil {
		return fmt.Errorf("failed to list tools for %s: %w", name, err)
	}

	for _, tool := range res.Tools {
		toolName := tool.Name

		var paramsMap map[string]any
		schemaBytes, _ := json.Marshal(tool.InputSchema)
		_ = json.Unmarshal(schemaBytes, &paramsMap)

		if paramsMap == nil {
			paramsMap = make(map[string]any)
		}
		if _, ok := paramsMap["type"]; !ok {
			paramsMap["type"] = "object"
		}
		if _, ok := paramsMap["properties"]; !ok {
			paramsMap["properties"] = map[string]any{}
		}

		registeredName := toolName
		if m.registry.Has(toolName) {
			registeredName = fmt.Sprintf("%s_%s", name, toolName)
			fmt.Fprintf(os.Stderr, "Warning: MCP tool %q from server %q conflicts with an existing tool; registering as %q\n", toolName, name, registeredName)
		}

		spec := tools.ToolSpec{
			Type: "function",
			Function: tools.FunctionSpec{
				Name:        registeredName,
				Description: tool.Description,
				Parameters:  paramsMap,
			},
		}

		// Closure copy
		clientObj := c
		mcpToolName := toolName

		m.registry.Register(spec, func(execCtx context.Context, args map[string]any) (string, error) {
			callReq := mcp.CallToolRequest{}
			callReq.Params.Name = mcpToolName
			callReq.Params.Arguments = args

			callCtx, cancel := context.WithTimeout(execCtx, 60*time.Second)
			defer cancel()

			callRes, err := clientObj.CallTool(callCtx, callReq)
			if err != nil {
				return "", fmt.Errorf("MCP tool execution failed: %w", err)
			}

			var out string
			for _, item := range callRes.Content {
				switch v := item.(type) {
				case mcp.TextContent:
					out += v.Text
				case *mcp.TextContent:
					out += v.Text
				default:
					b, _ := json.Marshal(item)
					out += string(b)
				}
			}

			if callRes.IsError {
				return out, fmt.Errorf("MCP tool error: %s", out)
			}

			return out, nil
		})
	}

	return nil
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, c := range m.clients {
		c.Close()
	}
	m.clients = make(map[string]mcpclient.MCPClient)
}
