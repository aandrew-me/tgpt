package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aandrew-me/tgpt/v2/src/tools"
)

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
	fmt.Println("\n▶ Server Connection Type:")
	fmt.Println("  1) Remote URL (HTTP / SSE endpoint)")
	fmt.Println("  2) Local Command (stdio subprocess, e.g. npx, uvx)")
	fmt.Print("Choose option [1/2] (default 1): ")
	connTypeChoice, _ := readInput(reader)
	connTypeChoice = strings.TrimSpace(connTypeChoice)

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

		fmt.Print("▶ Transport Type [auto / streamable-http / sse] (default auto): ")
		tType, _ := readInput(reader)
		tType = strings.TrimSpace(tType)
		if tType != "" && tType != "auto" {
			sc.Type = tType
		}

		fmt.Print("▶ Add Authorization Bearer Token / API Key? [y/N]: ")
		addAuth, _ := readInput(reader)
		addAuth = strings.TrimSpace(strings.ToLower(addAuth))
		if addAuth == "y" || addAuth == "yes" {
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

		fmt.Print("▶ Add additional custom HTTP headers? [y/N]: ")
		addHeaders, _ := readInput(reader)
		addHeaders = strings.TrimSpace(strings.ToLower(addHeaders))
		if addHeaders == "y" || addHeaders == "yes" {
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
		fmt.Print("Do you still want to save this server configuration? [y/N]: ")
		saveAnyway, _ := readInput(reader)
		saveAnyway = strings.TrimSpace(strings.ToLower(saveAnyway))
		if saveAnyway != "y" && saveAnyway != "yes" {
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
	cfg, _ := LoadConfig(resolvedPath)
	if cfg == nil {
		cfg = &Config{
			MCPServers: make(map[string]ServerConfig),
		}
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
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0644)
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
