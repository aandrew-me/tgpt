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
