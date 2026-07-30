package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	tgptclient "github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/tools"
	mcpclient "github.com/mark3labs/mcp-go/client"
	mcptransport "github.com/mark3labs/mcp-go/client/transport"
	"github.com/mark3labs/mcp-go/mcp"
)

type ServerConfig struct {
	Command string            `json:"command,omitempty"`
	Args    []string          `json:"args,omitempty"`
	Env     []string          `json:"env,omitempty"`
	URL     string            `json:"url,omitempty"`
	Type    string            `json:"type,omitempty"`
	Headers map[string]string `json:"headers,omitempty"`
}

type headerTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (h *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req2 := req.Clone(req.Context())
	for k, v := range h.headers {
		req2.Header.Set(k, os.ExpandEnv(v))
	}
	return h.base.RoundTrip(req2)
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

	var mcpClient *mcpclient.Client
	var err error

	if sc.URL != "" {
		httpClient := tgptclient.NewStandardHTTPClient(60)
		if len(sc.Headers) > 0 {
			baseTransport := httpClient.Transport
			if baseTransport == nil {
				baseTransport = http.DefaultTransport
			}
			httpClient.Transport = &headerTransport{
				base:    baseTransport,
				headers: sc.Headers,
			}
		}

		switch sc.Type {
		case "sse":
			mcpClient, err = mcpclient.NewSSEMCPClient(sc.URL, mcpclient.WithHTTPClient(httpClient))
			if err == nil {
				err = mcpClient.Start(ctx)
			}
		case "http", "streamable-http":
			mcpClient, err = mcpclient.NewStreamableHttpClient(sc.URL, mcptransport.WithHTTPBasicClient(httpClient))
			if err == nil {
				err = mcpClient.Start(ctx)
			}
		default:
			// Try Streamable HTTP first (modern MCP spec used by servers like Exa)
			mcpClient, err = mcpclient.NewStreamableHttpClient(sc.URL, mcptransport.WithHTTPBasicClient(httpClient))
			if err == nil {
				err = mcpClient.Start(ctx)
			}
			if err != nil {
				if mcpClient != nil {
					mcpClient.Close()
				}
				// Fall back to SSE transport if Streamable HTTP fails
				mcpClient, err = mcpclient.NewSSEMCPClient(sc.URL, mcpclient.WithHTTPClient(httpClient))
				if err == nil {
					err = mcpClient.Start(ctx)
				}
			}
		}
		if err != nil {
			if mcpClient != nil {
				mcpClient.Close()
			}
			return fmt.Errorf("failed to start MCP client for %s: %w", name, err)
		}
	} else if sc.Command != "" {
		mcpClient, err = mcpclient.NewStdioMCPClient(sc.Command, sc.Env, sc.Args...)
		if err != nil {
			return fmt.Errorf("failed to create stdio MCP client for %s: %w", name, err)
		}
		if err := mcpClient.Start(ctx); err != nil {
			mcpClient.Close()
			return fmt.Errorf("failed to start MCP client for %s: %w", name, err)
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

	_, err = mcpClient.Initialize(ctx, initReq)
	if err != nil {
		mcpClient.Close()
		return fmt.Errorf("failed to initialize MCP client for %s: %w", name, err)
	}

	m.clients[name] = mcpClient

	// List tools and register them
	listToolsReq := mcp.ListToolsRequest{}
	res, err := mcpClient.ListTools(ctx, listToolsReq)
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
		clientObj := mcpClient
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
