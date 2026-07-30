package tools

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRegistryExecute(t *testing.T) {
	r := NewRegistry()

	res, err := r.Execute(context.Background(), "read_directory", `{"path": "."}`)
	if err != nil {
		t.Fatalf("unexpected error executing read_directory: %v", err)
	}
	if res == "" {
		t.Fatal("expected non-empty output from read_directory")
	}
}

func TestReadFileTool(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	expected := "Hello, tgpt tools!"
	if err := os.WriteFile(filePath, []byte(expected), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	r := NewRegistry()
	res, err := r.Execute(context.Background(), "read_file", `{"path": "`+strings.ReplaceAll(filePath, `\`, `\\`)+`"}`)
	if err != nil {
		t.Fatalf("unexpected error executing read_file: %v", err)
	}
	if res != expected {
		t.Fatalf("expected %q, got %q", expected, res)
	}
}

func TestExecuteNonExistentTool(t *testing.T) {
	r := NewRegistry()
	_, err := r.Execute(context.Background(), "non_existent", `{}`)
	if err == nil {
		t.Fatal("expected error executing non-existent tool, got nil")
	}
}

func TestRegistryHas(t *testing.T) {
	r := NewRegistry()
	if !r.Has("read_file") {
		t.Fatal("expected registry to have built-in read_file tool")
	}
	if r.Has("non_existent_tool") {
		t.Fatal("expected registry to not have non_existent_tool")
	}
}

func TestExecuteCommandAutoExec(t *testing.T) {
	r := NewRegistry()
	ctx := context.WithValue(context.Background(), AutoExecKey, true)
	res, err := r.Execute(ctx, "execute_command", `{"command": "echo hello"}`)
	if err != nil {
		t.Fatalf("unexpected error executing execute_command: %v", err)
	}
	if !strings.Contains(res, "hello") {
		t.Fatalf("expected output to contain 'hello', got %q", res)
	}
}
