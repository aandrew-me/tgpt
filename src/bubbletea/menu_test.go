package bubbletea

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func TestSelectModelNavigation(t *testing.T) {
	m := SelectModel{
		Title:   "Test Title",
		Options: []string{"Yes", "No"},
		Cursor:  0,
	}

	updatedModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	sm := updatedModel.(SelectModel)
	if sm.Cursor != 1 {
		t.Errorf("expected cursor 1 after down key, got %d", sm.Cursor)
	}

	updatedModel, _ = sm.Update(tea.KeyPressMsg{Code: tea.KeyUp})
	sm = updatedModel.(SelectModel)
	if sm.Cursor != 0 {
		t.Errorf("expected cursor 0 after up key, got %d", sm.Cursor)
	}
}

func TestSelectModelHotkeys(t *testing.T) {
	m := SelectModel{
		Title:    "Confirm",
		Options:  []string{"Yes", "No"},
		Cursor:   1,
		Selected: -1,
	}

	updatedModel, _ := m.Update(tea.KeyPressMsg{Code: 'y'})
	sm := updatedModel.(SelectModel)
	if sm.Selected != 0 {
		t.Errorf("expected selected 0 for 'y' hotkey, got %d", sm.Selected)
	}

	updatedModel, _ = m.Update(tea.KeyPressMsg{Code: 'n'})
	sm = updatedModel.(SelectModel)
	if sm.Selected != 1 {
		t.Errorf("expected selected 1 for 'n' hotkey, got %d", sm.Selected)
	}
}

func TestSelectModelCancel(t *testing.T) {
	m := SelectModel{
		Title:   "Confirm",
		Options: []string{"Yes", "No"},
	}

	updatedModel, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEsc})
	sm := updatedModel.(SelectModel)
	if !sm.Canceled {
		t.Errorf("expected canceled to be true after esc key")
	}
}

func TestSelectModelInterrupt(t *testing.T) {
	m := SelectModel{
		Title:   "Confirm",
		Options: []string{"Yes", "No"},
	}

	updatedModel, _ := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
	sm := updatedModel.(SelectModel)
	if !sm.Interrupted {
		t.Errorf("expected interrupted to be true after ctrl+c key")
	}
}

func TestSelectModelView(t *testing.T) {
	m := SelectModel{
		Title:   "Confirm Action?",
		Options: []string{"Yes", "No"},
		Cursor:  0,
	}

	view := m.View()
	viewStr := view.Content
	if !strings.Contains(viewStr, "Confirm Action?") {
		t.Errorf("view output does not contain title: %s", viewStr)
	}
	if !strings.Contains(viewStr, "❯ Yes") {
		t.Errorf("view output does not contain highlighted cursor option: %s", viewStr)
	}
}

func TestSelectMenuEmpty(t *testing.T) {
	_, _, err := SelectMenu("Empty", []string{}, 0)
	if err == nil {
		t.Errorf("expected error when options are empty")
	}
}
