package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/aandrew-me/tgpt/v2/src/bubbletea"
	"github.com/aandrew-me/tgpt/v2/src/tools"
)

// SelectMenu runs an interactive selection menu using arrow keys and Enter.
// Returns the selected index, selected option string, or error if canceled.
func SelectMenu(title string, options []string, defaultIndex int) (int, string, error) {
	idx, opt, err := bubbletea.SelectMenu(title, options, defaultIndex)
	if errors.Is(err, bubbletea.ErrInterrupted) {
		bubbletea.RestoreTerminal()
		os.Exit(130)
	}
	return idx, opt, err
}

func ConfirmMenu(title string, defaultYes bool) (bool, error) {
	confirmed, err := bubbletea.ConfirmMenu(title, defaultYes)
	if errors.Is(err, bubbletea.ErrInterrupted) {
		bubbletea.RestoreTerminal()
		os.Exit(130)
	}
	return confirmed, err
}

// RemoveServerInteractive lists configured MCP servers in an interactive arrow-key menu
// and allows the user to select and remove one.
func RemoveServerInteractive(ctx context.Context, configPath string) error {
	resolvedPath := configPath
	if resolvedPath == "" {
		resolvedPath = "mcp_config.json"
		if homeDir, err := os.UserHomeDir(); err == nil {
			userConfig := filepath.Join(homeDir, ".config", "tgpt", "mcp_config.json")
			if _, err := os.Stat("mcp_config.json"); os.IsNotExist(err) {
				if _, err := os.Stat(userConfig); err == nil {
					resolvedPath = userConfig
				}
			}
		}
	}

	cfg, err := LoadConfig(resolvedPath)
	if err != nil {
		return fmt.Errorf("failed to load MCP config from %s: %w", resolvedPath, err)
	}

	if cfg == nil || len(cfg.MCPServers) == 0 {
		fmt.Printf("No MCP servers found in configuration file (%s).\n", resolvedPath)
		return nil
	}

	var serverNames []string
	for name := range cfg.MCPServers {
		serverNames = append(serverNames, name)
	}
	sort.Strings(serverNames)

	var displayItems []string
	for _, name := range serverNames {
		sc := cfg.MCPServers[name]
		var detail string
		if sc.URL != "" {
			detail = sc.URL
		} else if sc.Command != "" {
			detail = "stdio: " + sc.Command
			if len(sc.Args) > 0 {
				detail += " " + strings.Join(sc.Args, " ")
			}
		}
		if detail != "" {
			displayItems = append(displayItems, fmt.Sprintf("%s (%s)", name, detail))
		} else {
			displayItems = append(displayItems, name)
		}
	}

	serverNames = append(serverNames, "Cancel")
	displayItems = append(displayItems, "Cancel (abort without removing)")

	title := fmt.Sprintf("\n╭─ Remove MCP Server (%s)\n│ Select an MCP server to remove:", resolvedPath)
	idx, _, err := SelectMenu(title, displayItems, 0)
	if err != nil || idx < 0 || idx >= len(serverNames)-1 {
		fmt.Println("Aborted without removing.")
		return nil
	}

	selectedName := serverNames[idx]
	delete(cfg.MCPServers, selectedName)

	if err := SaveConfig(resolvedPath, cfg); err != nil {
		return fmt.Errorf("failed to save MCP config to %s: %w", resolvedPath, err)
	}

	fmt.Printf("\n✓ MCP Server %q successfully removed from %s\n", selectedName, resolvedPath)
	return nil
}

// AddServerInteractive prompts the user for server details, tests the connection,
// and saves the server configuration to the specified config file.
func AddServerInteractive(ctx context.Context, configPath string) error {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n╭─ Add New MCP Server")
	fmt.Println("│ Configure a new Model Context Protocol (MCP) server for tgpt.")
	fmt.Println("╰─────────────────────────────────────────────────────────────")

	// 1. Server Name
	fmt.Print("\n▶ Server name (e.g. firecrawl, github, filesystem): ")
	name, err := readInput(reader)
	if err != nil {
		return err
	}
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("server name cannot be empty")
	}

	// Determine destination config file path early
	resolvedPath := configPath
	if resolvedPath == "" {
		resolvedPath = "mcp_config.json"
		if homeDir, err := os.UserHomeDir(); err == nil {
			userConfig := filepath.Join(homeDir, ".config", "tgpt", "mcp_config.json")
			if _, err := os.Stat("mcp_config.json"); os.IsNotExist(err) {
				if _, err := os.Stat(userConfig); err == nil {
					resolvedPath = userConfig
				}
			}
		}
	}

	// 2. Connection Type
	connTypeChoice := "1"
	connOptions := []string{
		"Remote URL (HTTP / SSE endpoint)",
		"Local Command (stdio subprocess, e.g. npx, uvx)",
	}
	connIdx, _, err := SelectMenu("▶ Server Connection Type:", connOptions, 0)
	if err == nil {
		if connIdx == 1 {
			connTypeChoice = "2"
		}
	} else {
		fmt.Println("\n▶ Server Connection Type:")
		fmt.Println("  1) Remote URL (HTTP / SSE endpoint)")
		fmt.Println("  2) Local Command (stdio subprocess, e.g. npx, uvx)")
		fmt.Print("Choose option [1/2] (default 1): ")
		val, _ := readInput(reader)
		val = strings.TrimSpace(val)
		if val == "2" {
			connTypeChoice = "2"
		}
	}

	var sc ServerConfig

	if connTypeChoice == "2" {
		// Stdio Server
		fmt.Print("\n▶ Executable Command (e.g. npx, uvx, node): ")
		cmd, err := readInput(reader)
		if err != nil {
			return err
		}
		sc.Command = strings.TrimSpace(cmd)
		if sc.Command == "" {
			return fmt.Errorf("command cannot be empty for stdio server")
		}

		fmt.Print("▶ Command Arguments (space-separated, optional): ")
		argsStr, _ := readInput(reader)
		argsStr = strings.TrimSpace(argsStr)
		if argsStr != "" {
			sc.Args = parseArgs(argsStr)
		}

		fmt.Print("▶ Environment Variables (KEY=VALUE comma-separated, optional): ")
		envStr, _ := readInput(reader)
		envStr = strings.TrimSpace(envStr)
		if envStr != "" {
			parts := strings.Split(envStr, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					sc.Env = append(sc.Env, p)
				}
			}
		}
	} else {
		// Remote HTTP/SSE Server
		fmt.Print("\n▶ Server URL (e.g. https://mcp.firecrawl.dev/v2/mcp): ")
		urlStr, err := readInput(reader)
		if err != nil {
			return err
		}
		sc.URL = strings.TrimSpace(urlStr)
		if sc.URL == "" {
			return fmt.Errorf("server URL cannot be empty")
		}

		tOptions := []string{
			"auto (default - try Streamable HTTP, fallback to SSE)",
			"streamable-http",
			"sse",
		}
		tIdx, _, err := SelectMenu("▶ Transport Type:", tOptions, 0)
		if err == nil {
			if tIdx == 1 {
				sc.Type = "streamable-http"
			} else if tIdx == 2 {
				sc.Type = "sse"
			}
		} else {
			fmt.Print("▶ Transport Type [auto / streamable-http / sse] (default auto): ")
			tType, _ := readInput(reader)
			tType = strings.TrimSpace(tType)
			if tType != "" && tType != "auto" {
				sc.Type = tType
			}
		}

		addAuth, _ := ConfirmMenu("▶ Add Authorization Bearer Token / API Key?", false)
		if addAuth {
			fmt.Print("  Enter Bearer Token / API Key: ")
			token, _ := readInput(reader)
			token = strings.TrimSpace(token)
			if token != "" {
				if sc.Headers == nil {
					sc.Headers = make(map[string]string)
				}
				if !strings.HasPrefix(token, "Bearer ") {
					sc.Headers["Authorization"] = "Bearer " + token
				} else {
					sc.Headers["Authorization"] = token
				}
			}
		}

		addHeaders, _ := ConfirmMenu("▶ Add additional custom HTTP headers?", false)
		if addHeaders {
			for {
				fmt.Print("  Header Name (or press Enter to finish): ")
				hName, _ := readInput(reader)
				hName = strings.TrimSpace(hName)
				if hName == "" {
					break
				}
				fmt.Printf("  Header Value for %s: ", hName)
				hVal, _ := readInput(reader)
				hVal = strings.TrimSpace(hVal)
				if sc.Headers == nil {
					sc.Headers = make(map[string]string)
				}
				sc.Headers[hName] = hVal
			}
		}
	}

	// 3. Test Connection
	fmt.Printf("\n⏳ Testing connection to MCP server %q...\n", name)
	testRegistry := tools.NewRegistry()
	testMgr := NewManager(testRegistry)

	initErr := testMgr.InitServer(ctx, name, sc)
	if initErr != nil {
		fmt.Printf("\n⚠️  Connection test warning: %v\n", initErr)
		saveAnyway, _ := ConfirmMenu("Do you still want to save this server configuration?", false)
		if !saveAnyway {
			fmt.Println("Aborted without saving.")
			return nil
		}
	} else {
		allTools := testRegistry.ListSpecs()
		var toolNames []string
		for _, t := range allTools {
			toolNames = append(toolNames, t.Function.Name)
		}
		fmt.Printf("✓ Connection successful! Discovered %d tool(s): [%s]\n", len(toolNames), strings.Join(toolNames, ", "))
		testMgr.Close()
	}

	// 4. Save to Config File
	cfg, err := LoadConfig(resolvedPath)
	if err != nil {
		// File exists but couldn't be parsed; refuse to overwrite it.
		if _, statErr := os.Stat(resolvedPath); statErr == nil {
			return fmt.Errorf("existing config %s could not be parsed, refusing to overwrite: %w", resolvedPath, err)
		}
	}
	if cfg == nil {
		cfg = &Config{MCPServers: make(map[string]ServerConfig)}
	}
	if cfg.MCPServers == nil {
		cfg.MCPServers = make(map[string]ServerConfig)
	}

	cfg.MCPServers[name] = sc

	if err := SaveConfig(resolvedPath, cfg); err != nil {
		return fmt.Errorf("failed to save MCP config to %s: %w", resolvedPath, err)
	}

	fmt.Printf("\nMCP Server %q successfully saved to %s\n", name, resolvedPath)
	return nil
}

// SaveConfig writes the given Config struct to path formatted as indented JSON.
func SaveConfig(path string, cfg *Config) error {
	dir := filepath.Dir(path)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0600)
}

func readInput(reader *bufio.Reader) (string, error) {
	line, err := reader.ReadString('\n')
	if err != nil && line == "" {
		return "", err
	}
	return strings.TrimRight(line, "\r\n"), nil
}

func parseArgs(argsStr string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false
	quoteChar := rune(0)

	for _, ch := range argsStr {
		switch {
		case ch == '\'' || ch == '"':
			if inQuotes && ch == quoteChar {
				inQuotes = false
				quoteChar = 0
			} else if !inQuotes {
				inQuotes = true
				quoteChar = ch
			} else {
				current.WriteRune(ch)
			}
		case ch == ' ' || ch == '\t':
			if inQuotes {
				current.WriteRune(ch)
			} else if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

