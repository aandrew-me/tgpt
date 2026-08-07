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
	r.RegisterBuiltinTools()

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
	r.RegisterBuiltinTools()
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
	r.RegisterBuiltinTools()
	if !r.Has("read_file") {
		t.Fatal("expected registry to have built-in read_file tool")
	}
	if r.Has("non_existent_tool") {
		t.Fatal("expected registry to not have non_existent_tool")
	}
}

func TestExecuteCommandAutoExec(t *testing.T) {
	r := NewRegistry()
	r.RegisterBuiltinTools()
	ctx := context.WithValue(context.Background(), AutoExecKey, true)
	res, err := r.Execute(ctx, "execute_command", `{"command": "echo hello"}`)
	if err != nil {
		t.Fatalf("unexpected error executing execute_command: %v", err)
	}
	if !strings.Contains(res, "hello") {
		t.Fatalf("expected output to contain 'hello', got %q", res)
	}
}

func TestWriteFileTool(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "sub", "test_write.txt")
	expectedContent := "Hello from write_file tool!"

	r := NewRegistry()
	r.RegisterBuiltinTools()
	if !r.Has("write_file") {
		t.Fatal("expected registry to have write_file tool")
	}

	escapedPath := strings.ReplaceAll(filePath, `\`, `\\`)
	argsJSON := `{"path": "` + escapedPath + `", "content": "` + expectedContent + `"}`
	res, err := r.Execute(context.Background(), "write_file", argsJSON)
	if err != nil {
		t.Fatalf("unexpected error executing write_file: %v", err)
	}
	if !strings.Contains(res, "Successfully wrote") {
		t.Fatalf("expected success message, got %q", res)
	}

	readBack, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read back written file: %v", err)
	}
	if string(readBack) != expectedContent {
		t.Fatalf("expected %q, got %q", expectedContent, string(readBack))
	}
}

func TestRegisterIndividualTools(t *testing.T) {
	r := NewRegistry()
	r.RegisterBuiltinTools("web_search_exa", "read_file")

	if !r.Has("web_search_exa") {
		t.Error("expected registry to have web_search_exa")
	}
	if !r.Has("read_file") {
		t.Error("expected registry to have read_file")
	}
	if r.Has("web_search_firecrawl") {
		t.Error("expected registry to NOT have web_search_firecrawl")
	}
	if r.Has("execute_command") {
		t.Error("expected registry to NOT have execute_command")
	}
	if r.Has("read_directory") {
		t.Error("expected registry to NOT have read_directory")
	}
	if r.Has("web_fetch") {
		t.Error("expected registry to NOT have web_fetch")
	}
	if r.Has("write_file") {
		t.Error("expected registry to NOT have write_file")
	}
}

func TestParseToolList(t *testing.T) {
	tools, ok := ParseToolList("web_search_exa, read_file")
	if !ok || len(tools) != 2 || tools[0] != "web_search_exa" || tools[1] != "read_file" {
		t.Fatalf("unexpected ParseToolList result: %v, %v", tools, ok)
	}

	_, ok = ParseToolList("invalid_tool")
	if ok {
		t.Fatal("expected ParseToolList to return false for invalid_tool")
	}

	tools, ok = ParseToolList("all")
	if !ok || len(tools) != 1 || tools[0] != "all" {
		t.Fatalf("unexpected ParseToolList result for 'all': %v, %v", tools, ok)
	}
}

func TestPreConfirmAutoExec(t *testing.T) {
	ctx := context.WithValue(context.Background(), AutoExecKey, true)
	proceed, msg, err := PreConfirm(ctx, "execute_command", `{"command": "echo test"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proceed || msg != "" {
		t.Fatalf("expected proceed true with empty msg, got %v, %q", proceed, msg)
	}
}
