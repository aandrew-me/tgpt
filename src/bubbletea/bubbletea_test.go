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
