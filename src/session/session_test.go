package session

import (
	"testing"
	"time"

	"github.com/aandrew-me/tgpt/v2/src/structs"
)

func setupTestDir(t *testing.T) string {
	t.Helper()
	tempDir := t.TempDir()
	SetSessionsDir(tempDir)
	t.Cleanup(func() {
		SetSessionsDir("")
	})
	return tempDir
}

func TestValidateName(t *testing.T) {
	valid := []string{"work", "my-session", "session_1", "test.v2"}
	for _, name := range valid {
		if err := ValidateName(name); err != nil {
			t.Errorf("expected valid name %q, got error: %v", name, err)
		}
	}

	invalid := []string{"", "  ", ".", "..", "foo/bar", "foo\\bar", "session:1", "hello*"}
	for _, name := range invalid {
		if err := ValidateName(name); err == nil {
			t.Errorf("expected error for invalid name %q, got nil", name)
		}
	}
}

func TestSaveAndLoadRoundtrip(t *testing.T) {
	setupTestDir(t)

	sessionName := "test-roundtrip"
	messages := []any{
		structs.DefaultMessage{Role: "user", Content: "Hello AI"},
		structs.AssistantToolCallMessage{
			Role: "assistant",
			ToolCalls: []structs.ToolCall{
				{
					ID:   "call_123",
					Type: "function",
					Function: structs.ToolCallFunction{
						Name:      "read_file",
						Arguments: `{"path":"main.go"}`,
					},
				},
			},
		},
		structs.ToolMessage{
			Role:       "tool",
			ToolCallID: "call_123",
			Name:       "read_file",
			Content:    "package main",
		},
		structs.DefaultMessage{Role: "assistant", Content: "Here is your file"},
	}

	if err := Save(sessionName, messages); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	if !Exists(sessionName) {
		t.Fatalf("expected session %q to exist", sessionName)
	}

	loaded, err := Load(sessionName)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if len(loaded) != len(messages) {
		t.Fatalf("expected %d messages, got %d", len(messages), len(loaded))
	}

	// Verify message types and contents
	m0, ok := loaded[0].(structs.DefaultMessage)
	if !ok || m0.Role != "user" || m0.Content != "Hello AI" {
		t.Errorf("m0 mismatch: %+v", loaded[0])
	}

	m1, ok := loaded[1].(structs.AssistantToolCallMessage)
	if !ok || len(m1.ToolCalls) != 1 || m1.ToolCalls[0].Function.Name != "read_file" {
		t.Errorf("m1 mismatch: %+v", loaded[1])
	}

	m2, ok := loaded[2].(structs.ToolMessage)
	if !ok || m2.ToolCallID != "call_123" || m2.Content != "package main" {
		t.Errorf("m2 mismatch: %+v", loaded[2])
	}

	m3, ok := loaded[3].(structs.DefaultMessage)
	if !ok || m3.Role != "assistant" || m3.Content != "Here is your file" {
		t.Errorf("m3 mismatch: %+v", loaded[3])
	}
}

func TestListAndGetLatest(t *testing.T) {
	setupTestDir(t)

	// Initially empty
	list, err := List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(list))
	}

	if _, err := GetLatest(); err == nil {
		t.Fatal("expected error on GetLatest with no sessions, got nil")
	}

	// Create session A
	_ = Save("session-a", []any{
		structs.DefaultMessage{Role: "user", Content: "A"},
	})
	time.Sleep(10 * time.Millisecond)

	// Create session B (newer)
	_ = Save("session-b", []any{
		structs.DefaultMessage{Role: "user", Content: "B1"},
		structs.DefaultMessage{Role: "assistant", Content: "B2"},
	})

	list, err = List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(list))
	}

	// Newest first
	if list[0].Name != "session-b" || list[0].MessageCount != 2 {
		t.Errorf("expected session-b first with count 2, got %+v", list[0])
	}
	if list[1].Name != "session-a" || list[1].MessageCount != 1 {
		t.Errorf("expected session-a second with count 1, got %+v", list[1])
	}

	latest, err := GetLatest()
	if err != nil || latest != "session-b" {
		t.Fatalf("expected GetLatest = session-b, got %q (err: %v)", latest, err)
	}
}

func TestDelete(t *testing.T) {
	setupTestDir(t)

	_ = Save("to-delete", []any{structs.DefaultMessage{Role: "user", Content: "bye"}})
	if !Exists("to-delete") {
		t.Fatal("expected session to exist")
	}

	if err := Delete("to-delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}
	if Exists("to-delete") {
		t.Fatal("expected session to no longer exist")
	}

	if err := Delete("to-delete"); err == nil {
		t.Fatal("expected error when deleting non-existent session, got nil")
	}
}

func TestExtractHistory(t *testing.T) {
	messages := []any{
		structs.DefaultMessage{Role: "user", Content: "first prompt"},
		structs.DefaultMessage{Role: "assistant", Content: "first response"},
		structs.DefaultMessage{Role: "user", Content: "second prompt"},
	}
	history := ExtractHistory(messages)
	if len(history) != 2 || history[0] != "first prompt" || history[1] != "second prompt" {
		t.Fatalf("unexpected history: %#v", history)
	}
}
