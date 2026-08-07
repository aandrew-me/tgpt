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
		t.Fatalf("unexpected error executing write_file for new file: %v", err)
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

func TestPreConfirmNewFileNoPrompt(t *testing.T) {
	dir := t.TempDir()
	newPath := filepath.Join(dir, "non_existent_file.txt")
	escapedPath := strings.ReplaceAll(newPath, `\`, `\\`)
	proceed, msg, err := PreConfirm(context.Background(), "write_file", `{"path": "`+escapedPath+`"}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !proceed || msg != "" {
		t.Fatalf("expected proceed true without confirmation for new file, got %v, %q", proceed, msg)
	}
}

func TestWriteFileAppendMode(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "append_test.txt")
	if err := os.WriteFile(filePath, []byte("Hello "), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r := NewRegistry()
	r.RegisterBuiltinTools()

	escapedPath := strings.ReplaceAll(filePath, `\`, `\\`)
	ctx := context.WithValue(context.Background(), AutoExecKey, true)
	argsJSON := `{"path": "` + escapedPath + `", "content": "world!", "append": true}`

	res, err := r.Execute(ctx, "write_file", argsJSON)
	if err != nil {
		t.Fatalf("unexpected error executing write_file in append mode: %v", err)
	}
	if !strings.Contains(res, "Successfully appended") {
		t.Fatalf("expected append success message, got %q", res)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read back appended file: %v", err)
	}
	if string(content) != "Hello world!" {
		t.Fatalf("expected 'Hello world!', got %q", string(content))
	}
}

func TestEditFileTool(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "edit_test.txt")
	initialContent := "func foo() {\n\treturn 1\n}\n"
	if err := os.WriteFile(filePath, []byte(initialContent), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r := NewRegistry()
	r.RegisterBuiltinTools()
	if !r.Has("edit_file") {
		t.Fatal("expected registry to have edit_file tool")
	}

	escapedPath := strings.ReplaceAll(filePath, `\`, `\\`)
	ctx := context.WithValue(context.Background(), AutoExecKey, true)

	// Successful edit
	argsJSON := `{"path": "` + escapedPath + `", "old_content": "return 1", "new_content": "return 42"}`
	res, err := r.Execute(ctx, "edit_file", argsJSON)
	if err != nil {
		t.Fatalf("unexpected error executing edit_file: %v", err)
	}
	if !strings.Contains(res, "Successfully edited") {
		t.Fatalf("expected edit success message, got %q", res)
	}

	readBack, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("failed to read back edited file: %v", err)
	}
	expected := "func foo() {\n\treturn 42\n}\n"
	if string(readBack) != expected {
		t.Fatalf("expected %q, got %q", expected, string(readBack))
	}

	// Target old_content not found error
	notFoundJSON := `{"path": "` + escapedPath + `", "old_content": "non_existent_code", "new_content": "something"}`
	_, err = r.Execute(ctx, "edit_file", notFoundJSON)
	if err == nil || !strings.Contains(err.Error(), "old_content not found") {
		t.Fatalf("expected 'old_content not found' error, got %v", err)
	}

	// Multiple matches error
	multiContent := "foo\nfoo\n"
	if err := os.WriteFile(filePath, []byte(multiContent), 0644); err != nil {
		t.Fatalf("failed to write multi-match file: %v", err)
	}
	multiJSON := `{"path": "` + escapedPath + `", "old_content": "foo", "new_content": "bar"}`
	_, err = r.Execute(ctx, "edit_file", multiJSON)
	if err == nil || !strings.Contains(err.Error(), "appears 2 times") {
		t.Fatalf("expected multiple matches error, got %v", err)
	}

	// Identical old_content and new_content error
	identicalJSON := `{"path": "` + escapedPath + `", "old_content": "foo", "new_content": "foo"}`
	_, err = r.Execute(ctx, "edit_file", identicalJSON)
	if err == nil || !strings.Contains(err.Error(), "identical") {
		t.Fatalf("expected identical content error, got %v", err)
	}

	// Non-existent file edit error
	missingPath := filepath.Join(dir, "does_not_exist.txt")
	escapedMissing := strings.ReplaceAll(missingPath, `\`, `\\`)
	missingJSON := `{"path": "` + escapedMissing + `", "old_content": "a", "new_content": "b"}`

	// PreConfirm should return proceed true without prompting for missing file
	proceed, msg, confirmErr := PreConfirm(context.Background(), "edit_file", missingJSON)
	if confirmErr != nil || !proceed || msg != "" {
		t.Fatalf("expected PreConfirm to skip prompt for missing file, got proceed=%v, msg=%q, err=%v", proceed, msg, confirmErr)
	}

	_, err = r.Execute(ctx, "edit_file", missingJSON)
	if err == nil || !strings.Contains(err.Error(), "failed to read file for editing") {
		t.Fatalf("expected failed to read file error, got %v", err)
	}
}
