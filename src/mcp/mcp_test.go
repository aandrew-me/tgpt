package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/aandrew-me/tgpt/v2/src/tools"
)

func TestLoadConfigNonExistent(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err == nil {
		t.Fatal("expected error loading non-existent config file, got nil")
	}
	if cfg != nil {
		t.Fatalf("expected nil config, got %#v", cfg)
	}
}

func TestLoadConfigValid(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp_config.json")
	content := `{
		"mcpServers": {
			"test-server": {
				"command": "echo",
				"args": ["hello"]
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || len(cfg.MCPServers) != 1 {
		t.Fatalf("expected 1 mcp server, got %#v", cfg)
	}
	sc, ok := cfg.MCPServers["test-server"]
	if !ok || sc.Command != "echo" || len(sc.Args) != 1 || sc.Args[0] != "hello" {
		t.Fatalf("unexpected server config: %#v", sc)
	}
}

func TestNewManagerDefaults(t *testing.T) {
	mgr := NewManager(nil)
	if mgr == nil {
		t.Fatal("expected non-nil Manager")
	}
	if mgr.registry != tools.DefaultRegistry {
		t.Fatal("expected default registry to be used when nil passed")
	}
}

func TestInitServerInvalid(t *testing.T) {
	mgr := NewManager(tools.NewRegistry())
	err := mgr.InitServer(t.Context(), "invalid", ServerConfig{})
	if err == nil {
		t.Fatal("expected error for empty server config")
	}
}

func TestLoadConfigWithHeaders(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp_config.json")
	content := `{
		"mcpServers": {
			"remote-server": {
				"url": "https://example.com/mcp",
				"headers": {
					"Authorization": "Bearer test-token",
					"X-Custom-Header": "custom-value"
				}
			}
		}
	}`
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	sc, ok := cfg.MCPServers["remote-server"]
	if !ok {
		t.Fatalf("expected remote-server config")
	}
	if sc.Headers["Authorization"] != "Bearer test-token" || sc.Headers["X-Custom-Header"] != "custom-value" {
		t.Fatalf("unexpected headers: %#v", sc.Headers)
	}
}

func TestSaveConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "sub", "mcp_config.json")
	cfg := &Config{
		MCPServers: map[string]ServerConfig{
			"test-srv": {
				Command: "npx",
				Args:    []string{"-y", "mcp-package"},
				Env:     []string{"ENV1=VAL1"},
			},
		},
	}

	if err := SaveConfig(configPath, cfg); err != nil {
		t.Fatalf("unexpected error saving config: %v", err)
	}

	loaded, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("failed to load saved config: %v", err)
	}

	sc, ok := loaded.MCPServers["test-srv"]
	if !ok || sc.Command != "npx" || len(sc.Args) != 2 || len(sc.Env) != 1 {
		t.Fatalf("unexpected loaded config: %#v", loaded)
	}
}

func TestParseArgs(t *testing.T) {
	parsed := parseArgs(`-y "@modelcontextprotocol/server-filesystem" "/path with space"`)
	expected := []string{"-y", "@modelcontextprotocol/server-filesystem", "/path with space"}
	if len(parsed) != len(expected) {
		t.Fatalf("expected %d args, got %d: %#v", len(expected), len(parsed), parsed)
	}
	for i, arg := range parsed {
		if arg != expected[i] {
			t.Errorf("arg[%d] expected %q, got %q", i, expected[i], arg)
		}
	}
}

func TestRemoveServerInteractiveEmptyConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp_config.json")
	if err := SaveConfig(configPath, &Config{MCPServers: make(map[string]ServerConfig)}); err != nil {
		t.Fatalf("unexpected error saving config: %v", err)
	}

	err := RemoveServerInteractive(t.Context(), configPath)
	if err != nil {
		t.Fatalf("expected no error when removing from empty config, got: %v", err)
	}
}

func TestLoadConfigDefaultPath(t *testing.T) {
	dir := t.TempDir()
	origWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}
	t.Cleanup(func() {
		os.Chdir(origWd)
	})

	content := `{
		"mcpServers": {
			"default-server": {
				"command": "echo",
				"args": ["default"]
			}
		}
	}`
	if err := os.WriteFile("mcp_config.json", []byte(content), 0644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	cfg, err := LoadConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg == nil || len(cfg.MCPServers) != 1 {
		t.Fatalf("expected 1 mcp server from default path, got %#v", cfg)
	}
	if _, ok := cfg.MCPServers["default-server"]; !ok {
		t.Fatalf("expected default-server in config")
	}
}


