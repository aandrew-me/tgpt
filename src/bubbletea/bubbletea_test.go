package bubbletea

import (
	"runtime"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestModelCtrlZ(t *testing.T) {
	preprompt := ""
	loop := true
	lastResponse := ""
	userInput := ""
	m := InitialModel(&preprompt, &loop, &lastResponse, &userInput)

	_, cmd := m.Update(tea.KeyPressMsg{Code: 'z', Mod: tea.ModCtrl})
	if runtime.GOOS != "windows" {
		if cmd == nil {
			t.Errorf("expected suspend command for ctrl+z on non-windows, got nil")
		}
	} else {
		if cmd != nil {
			t.Errorf("expected nil command for ctrl+z on windows, got %v", cmd)
		}
	}
}

func TestInputModelHistoryNavigation(t *testing.T) {
	history := []string{"first prompt", "second prompt"}
	m := InitialInputModel("╰─> ", history)

	// Simulate pressing Up key -> should show "second prompt"
	updatedModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	im := updatedModel.(InputModel)
	if im.textinput.Value() != "second prompt" {
		t.Errorf("expected 'second prompt', got %q", im.textinput.Value())
	}

	// Press Up key again -> should show "first prompt"
	updatedModel, _ = im.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	im = updatedModel.(InputModel)
	if im.textinput.Value() != "first prompt" {
		t.Errorf("expected 'first prompt', got %q", im.textinput.Value())
	}

	// Press Down key -> should show "second prompt"
	updatedModel, _ = im.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	im = updatedModel.(InputModel)
	if im.textinput.Value() != "second prompt" {
		t.Errorf("expected 'second prompt', got %q", im.textinput.Value())
	}

	// Press Down key -> should show empty original input
	updatedModel, _ = im.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	im = updatedModel.(InputModel)
	if im.textinput.Value() != "" {
		t.Errorf("expected '', got %q", im.textinput.Value())
	}
}

func TestInputModelSubmitAndCancel(t *testing.T) {
	m := InitialInputModel("╰─> ", nil)
	m.textinput.SetValue("test input")

	// Test Enter key
	updatedModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	im := updatedModel.(InputModel)
	if im.Value != "test input" {
		t.Errorf("expected 'test input', got %q", im.Value)
	}
	if im.Canceled {
		t.Errorf("expected Canceled to be false on enter")
	}

	// Test Ctrl+C key
	m = InitialInputModel("╰─> ", nil)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	im = updatedModel.(InputModel)
	if !im.Canceled {
		t.Errorf("expected Canceled to be true on Ctrl+C")
	}

	// Test Ctrl+D on empty input
	m = InitialInputModel("╰─> ", nil)
	updatedModel, _ = m.Update(tea.KeyPressMsg{Code: 'd', Mod: tea.ModCtrl})
	im = updatedModel.(InputModel)
	if !im.Canceled {
		t.Errorf("expected Canceled to be true on Ctrl+D on empty input")
	}
}
