package bubbletea

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	tea "charm.land/bubbletea/v2"
)

var (
	ErrInterrupted = errors.New("interrupted")
	ErrCanceled    = errors.New("selection canceled")
)

func RestoreTerminal() {
	if runtime.GOOS != "windows" {
		rawModeOff := exec.Command("stty", "-raw", "echo")
		rawModeOff.Stdin = os.Stdin
		_ = rawModeOff.Run()
	}
}

type SelectModel struct {
	Title       string
	Options     []string
	Cursor      int
	Selected    int
	Canceled    bool
	Interrupted bool
}

func (m SelectModel) Init() tea.Cmd {
	return nil
}

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Interrupted = true
			return m, tea.Quit
		case "esc", "q":
			m.Canceled = true
			return m, tea.Quit
		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
			} else {
				m.Cursor = len(m.Options) - 1
			}
		case "down", "j":
			if m.Cursor < len(m.Options)-1 {
				m.Cursor++
			} else {
				m.Cursor = 0
			}
		case "y", "Y":
			for i, opt := range m.Options {
				lower := strings.ToLower(opt)
				if lower == "yes" || strings.HasPrefix(lower, "yes ") || strings.HasPrefix(lower, "yes (") {
					m.Selected = i
					return m, tea.Quit
				}
			}
		case "n", "N":
			for i, opt := range m.Options {
				lower := strings.ToLower(opt)
				if lower == "no" || strings.HasPrefix(lower, "no ") || strings.HasPrefix(lower, "no (") {
					m.Selected = i
					return m, tea.Quit
				}
			}
		case "enter", " ":
			m.Selected = m.Cursor
			return m, tea.Quit
		}
	}
	return m, nil
}

func (m SelectModel) View() tea.View {
	var s strings.Builder
	if m.Title != "" {
		s.WriteString(m.Title + "\n")
	}
	for i, option := range m.Options {
		if m.Cursor == i {
			s.WriteString(fmt.Sprintf("  \033[36m❯ %s\033[0m\n", option))
		} else {
			s.WriteString(fmt.Sprintf("    %s\n", option))
		}
	}
	s.WriteString("\n\033[90m(Use ↑/↓ arrow keys to select, Enter to confirm, Esc to cancel)\033[0m\n")
	return tea.NewView(s.String())
}

// SelectMenu runs an interactive selection menu using arrow keys and Enter.
// Returns the selected index, selected option string, or error if canceled or interrupted.
func SelectMenu(title string, options []string, defaultIndex int) (int, string, error) {
	if len(options) == 0 {
		return -1, "", fmt.Errorf("no options provided")
	}
	if defaultIndex < 0 || defaultIndex >= len(options) {
		defaultIndex = 0
	}

	m := SelectModel{
		Title:    title,
		Options:  options,
		Cursor:   defaultIndex,
		Selected: -1,
	}

	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		// Non-interactive / pipe fallback
		return defaultIndex, options[defaultIndex], nil
	}

	res, ok := finalModel.(SelectModel)
	if !ok {
		return -1, "", fmt.Errorf("selection failed")
	}
	if res.Interrupted {
		return -1, "", ErrInterrupted
	}
	if res.Canceled || res.Selected < 0 {
		return -1, "", ErrCanceled
	}

	return res.Selected, res.Options[res.Selected], nil
}

// ConfirmMenu displays an interactive Yes/No selector.
// Returns true if "Yes" was selected, false if "No" or canceled.
func ConfirmMenu(title string, defaultYes bool) (bool, error) {
	defaultIndex := 0
	if !defaultYes {
		defaultIndex = 1
	}

	// Ensure prompt does not end with redundant [y/n]
	cleanTitle := strings.TrimRight(title, " \t:")
	if strings.HasSuffix(cleanTitle, "[y/n]") || strings.HasSuffix(cleanTitle, "[y/N]") {
		cleanTitle = strings.TrimSpace(cleanTitle[:strings.LastIndex(cleanTitle, "[")])
	}

	idx, _, err := SelectMenu(cleanTitle, []string{"Yes", "No"}, defaultIndex)
	if err != nil {
		return false, err
	}
	return idx == 0, nil
}
