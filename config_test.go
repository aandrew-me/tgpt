package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Create a temp config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.txt")
	content := `# This is a comment
				AI_PROVIDER=deepseek
				TEST_KEY_FROM_CONFIG=hello
				# Another comment
				EXISTING_ENV_VAR=new_val
				`
	err := os.WriteFile(configPath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("failed to write temp config file: %v", err)
	}

	os.Setenv("EXISTING_ENV_VAR", "original_val")
	defer os.Unsetenv("EXISTING_ENV_VAR")
	defer os.Unsetenv("AI_PROVIDER")
	defer os.Unsetenv("TEST_KEY_FROM_CONFIG")

	loadConfig(configPath)

	if val := os.Getenv("AI_PROVIDER"); val != "deepseek" {
		t.Errorf("expected AI_PROVIDER=deepseek, got %q", val)
	}

	if val := os.Getenv("TEST_KEY_FROM_CONFIG"); val != "hello" {
		t.Errorf("expected TEST_KEY_FROM_CONFIG=hello, got %q", val)
	}

	if val := os.Getenv("EXISTING_ENV_VAR"); val != "original_val" {
		t.Errorf("expected EXISTING_ENV_VAR to remain original_val, got %q", val)
	}
}
