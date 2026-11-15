package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/fatih/color"
)

var (
	blue              = color.New(color.FgBlue)
	customSessionsDir = ""
)

// SetSessionsDir overrides the default sessions directory (used for testing).
func SetSessionsDir(dir string) {
	customSessionsDir = dir
}

type SessionInfo struct {
	Name         string
	MessageCount int
	UpdatedAt    time.Time
}

type diskSession struct {
	Name      string            `json:"name"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
	Messages  []json.RawMessage `json:"messages"`
}

// ValidateName ensures session names are non-empty and safe for filesystem paths.
func ValidateName(name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return errors.New("session name cannot be empty")
	}
	if name == "." || name == ".." {
		return errors.New("invalid session name")
	}
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '-' && r != '_' && r != '.' {
			return fmt.Errorf("invalid character %q in session name (allowed: letters, digits, '-', '_', '.')", r)
		}
	}
	return nil
}

// GetSessionsDir returns the directory where sessions are stored (~/.config/tgpt/sessions).
func GetSessionsDir() (string, error) {
	if customSessionsDir != "" {
		return customSessionsDir, nil
	}
	homeDir, err := os.UserHomeDir()
	if err != nil || homeDir == "" {
		return filepath.Join(".", ".tgpt", "sessions"), nil
	}
	return filepath.Join(homeDir, ".config", "tgpt", "sessions"), nil
}

func filePath(name string) (string, error) {
	if err := ValidateName(name); err != nil {
		return "", err
	}
	dir, err := GetSessionsDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, name+".json"), nil
}

// Exists checks if a session exists on disk.
func Exists(name string) bool {
	path, err := filePath(name)
	if err != nil {
		return false
	}
	_, err = os.Stat(path)
	return err == nil
}

// Save atomically writes session messages to ~/.config/tgpt/sessions/<name>.json.
func Save(name string, messages []any) error {
	path, err := filePath(name)
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	createdAt := time.Now()
	if existingBytes, err := os.ReadFile(path); err == nil {
		var prev struct {
			CreatedAt time.Time `json:"created_at"`
		}
		if json.Unmarshal(existingBytes, &prev) == nil && !prev.CreatedAt.IsZero() {
			createdAt = prev.CreatedAt
		}
	}

	data := struct {
		Name      string    `json:"name"`
		CreatedAt time.Time `json:"created_at"`
		UpdatedAt time.Time `json:"updated_at"`
		Messages  []any     `json:"messages"`
	}{
		Name:      name,
		CreatedAt: createdAt,
		UpdatedAt: time.Now(),
		Messages:  messages,
	}

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	// Write to temp file then rename for atomic replacement.
	tempPath := fmt.Sprintf("%s.%d.tmp", path, time.Now().UnixNano())
	if err := os.WriteFile(tempPath, bytes, 0600); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}

// Load reads and parses previous messages for a session.
func Load(name string) ([]any, error) {
	path, err := filePath(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var disk diskSession
	if err := json.Unmarshal(data, &disk); err != nil {
		return nil, fmt.Errorf("corrupted session data: %w", err)
	}

	var messages []any
	for _, raw := range disk.Messages {
		var probe struct {
			Role       string             `json:"role"`
			ToolCalls  []structs.ToolCall `json:"tool_calls"`
			ToolCallID string             `json:"tool_call_id"`
		}
		if err := json.Unmarshal(raw, &probe); err == nil {
			if len(probe.ToolCalls) > 0 {
				var msg structs.AssistantToolCallMessage
				if err := json.Unmarshal(raw, &msg); err == nil {
					messages = append(messages, msg)
					continue
				}
			}
			if probe.ToolCallID != "" || probe.Role == "tool" {
				var msg structs.ToolMessage
				if err := json.Unmarshal(raw, &msg); err == nil {
					messages = append(messages, msg)
					continue
				}
			}
		}
		var msg structs.DefaultMessage
		if err := json.Unmarshal(raw, &msg); err == nil {
			messages = append(messages, msg)
		}
	}

	return messages, nil
}

// Delete removes a saved session file.
func Delete(name string) error {
	path, err := filePath(name)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("session %q does not exist", name)
		}
		return err
	}
	return nil
}

// List returns all saved sessions sorted by most recently modified first.
func List() ([]SessionInfo, error) {
	dir, err := GetSessionsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var sessions []SessionInfo
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		path := filepath.Join(dir, entry.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var parsed diskSession
		if err := json.Unmarshal(data, &parsed); err != nil {
			continue
		}
		updatedAt := parsed.UpdatedAt
		if updatedAt.IsZero() {
			if fi, err := entry.Info(); err == nil {
				updatedAt = fi.ModTime()
			}
		}
		sessions = append(sessions, SessionInfo{
			Name:         name,
			MessageCount: len(parsed.Messages),
			UpdatedAt:    updatedAt,
		})
	}

	sort.Slice(sessions, func(i, j int) bool {
		return sessions[i].UpdatedAt.After(sessions[j].UpdatedAt)
	})
	return sessions, nil
}

// GetLatest returns the name of the most recently updated session.
func GetLatest() (string, error) {
	sessions, err := List()
	if err != nil {
		return "", err
	}
	if len(sessions) == 0 {
		return "", errors.New("no saved sessions found")
	}
	return sessions[0].Name, nil
}

// Replay prints previous user and assistant messages to stdout.
func Replay(messages []any) {
	for _, raw := range messages {
		switch m := raw.(type) {
		case structs.DefaultMessage:
			if m.Role == "user" {
				blue.Println("╭─ You")
				blue.Print("╰─> ")
				fmt.Println(m.Content)
			} else if m.Role == "assistant" && m.Content != "" {
				fmt.Println(m.Content)
				fmt.Println()
			}
		}
	}
}

// ExtractHistory extracts user queries for prompt readline history navigation.
func ExtractHistory(messages []any) []string {
	var history []string
	for _, raw := range messages {
		if m, ok := raw.(structs.DefaultMessage); ok && m.Role == "user" && m.Content != "" {
			history = append(history, m.Content)
		}
	}
	return history
}
