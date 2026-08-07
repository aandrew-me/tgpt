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

func TestGrepTool(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "file1.go")
	file2 := filepath.Join(dir, "file2.txt")
	if err := os.WriteFile(file1, []byte("package main\nfunc Hello() {\n\tprintln(\"world\")\n}\n"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("Hello world in txt file\nfunc Other() {}\n"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}

	r := NewRegistry()
	r.RegisterBuiltinTools()
	if !r.Has("grep") {
		t.Fatal("expected registry to have grep tool")
	}

	escapedDir := strings.ReplaceAll(dir, `\`, `\\`)
	ctx := context.Background()

	// Search regex pattern across dir
	argsJSON := `{"path": "` + escapedDir + `", "pattern": "func \\w+\\(\\)"}`
	res, err := r.Execute(ctx, "grep", argsJSON)
	if err != nil {
		t.Fatalf("unexpected error executing grep: %v", err)
	}
	if !strings.Contains(res, "func Hello()") || !strings.Contains(res, "func Other()") {
		t.Fatalf("expected matches for func Hello() and func Other(), got: %q", res)
	}

	// Search with include filter
	includeJSON := `{"path": "` + escapedDir + `", "pattern": "func", "include": "*.go"}`
	res, err = r.Execute(ctx, "grep", includeJSON)
	if err != nil {
		t.Fatalf("unexpected error executing grep with include: %v", err)
	}
	if !strings.Contains(res, "file1.go") || strings.Contains(res, "file2.txt") {
		t.Fatalf("expected match only in file1.go, got: %q", res)
	}

	// No matches case
	noMatchJSON := `{"path": "` + escapedDir + `", "pattern": "nonexistent_pattern"}`
	res, err = r.Execute(ctx, "grep", noMatchJSON)
	if err != nil {
		t.Fatalf("unexpected error executing grep: %v", err)
	}
	if res != "No matches found." {
		t.Fatalf("expected 'No matches found.', got: %q", res)
	}

	// Binary file skip test
	binaryFile := filepath.Join(dir, "binary.bin")
	binaryData := append([]byte("PNG image data\x00with NUL byte\n"), []byte("MATCH_STRING")...)
	if err := os.WriteFile(binaryFile, binaryData, 0644); err != nil {
		t.Fatalf("failed to write binary file: %v", err)
	}
	binaryJSON := `{"path": "` + escapedDir + `", "pattern": "MATCH_STRING"}`
	res, err = r.Execute(ctx, "grep", binaryJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(res, "binary.bin") {
		t.Fatalf("expected binary file to be skipped, got matches: %q", res)
	}

	// Long line test (> 64KB)
	longFile := filepath.Join(dir, "longline.txt")
	longLine := strings.Repeat("a", 1000) + "TARGET_KEYWORD" + strings.Repeat("b", 70000) + "\n"
	if err := os.WriteFile(longFile, []byte(longLine), 0644); err != nil {
		t.Fatalf("failed to write long line file: %v", err)
	}
	longJSON := `{"path": "` + escapedDir + `", "pattern": "TARGET_KEYWORD"}`
	res, err = r.Execute(ctx, "grep", longJSON)
	if err != nil {
		t.Fatalf("unexpected error executing grep for long line: %v", err)
	}
	if !strings.Contains(res, "longline.txt") || !strings.Contains(res, "TARGET_KEYWORD") {
		t.Fatalf("expected match in long line file, got: %q", res)
	}
}

func TestGlobTool(t *testing.T) {
	dir := t.TempDir()
	file1 := filepath.Join(dir, "app.go")
	file2 := filepath.Join(dir, "config.json")
	subDir := filepath.Join(dir, "src")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("failed to create subDir: %v", err)
	}
	file3 := filepath.Join(subDir, "helper.go")

	if err := os.WriteFile(file1, []byte("package main"), 0644); err != nil {
		t.Fatalf("failed to write file1: %v", err)
	}
	if err := os.WriteFile(file2, []byte("{}"), 0644); err != nil {
		t.Fatalf("failed to write file2: %v", err)
	}
	if err := os.WriteFile(file3, []byte("package src"), 0644); err != nil {
		t.Fatalf("failed to write file3: %v", err)
	}

	r := NewRegistry()
	r.RegisterBuiltinTools()
	if !r.Has("glob") {
		t.Fatal("expected registry to have glob tool")
	}

	escapedDir := strings.ReplaceAll(dir, `\`, `\\`)
	ctx := context.Background()

	// Glob pattern match *.go
	argsJSON := `{"path": "` + escapedDir + `", "pattern": "*.go"}`
	res, err := r.Execute(ctx, "glob", argsJSON)
	if err != nil {
		t.Fatalf("unexpected error executing glob: %v", err)
	}
	if !strings.Contains(res, "app.go") || strings.Contains(res, "config.json") {
		t.Fatalf("expected match only for app.go, got: %q", res)
	}

	// Path-scoped glob match (src/*.go should match src/helper.go but NOT root app.go)
	scopedJSON := `{"path": "` + escapedDir + `", "pattern": "src/*.go"}`
	res, err = r.Execute(ctx, "glob", scopedJSON)
	if err != nil {
		t.Fatalf("unexpected error executing scoped glob: %v", err)
	}
	if !strings.Contains(res, "helper.go") || strings.Contains(res, "app.go") {
		t.Fatalf("expected scoped match for src/helper.go, got: %q", res)
	}

	// No match case
	noMatchJSON := `{"path": "` + escapedDir + `", "pattern": "*.txt"}`
	res, err = r.Execute(ctx, "glob", noMatchJSON)
	if err != nil {
		t.Fatalf("unexpected error executing glob: %v", err)
	}
	if res != "No matching files found." {
		t.Fatalf("expected 'No matching files found.', got: %q", res)
	}
}
