package bubbletea

import (
	"bufio"
	"fmt"
	"os"
	"runtime"
	"strings"

	"charm.land/bubbles/v2/textarea"
	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	"github.com/atotto/clipboard"
	"github.com/olekukonko/ts"
)

type model struct {
	textarea textarea.Model
	err      error
}

type errMsg error

var loop *bool
var userInput *string
var lastResp *string

func InitialModel(preprompt *string, programLoop *bool, lastResponse *string, userInputPtr *string) model {
	loop = programLoop
	lastResp = lastResponse
	userInput = userInputPtr

	size, _ := ts.GetSize()
	termWidth := size.Col()
	ti := textarea.New()
	ti.SetWidth(termWidth)
	ti.CharLimit = 200000
	ti.ShowLineNumbers = false
	ti.Placeholder = "Enter your prompt"
	ti.SetValue(*preprompt)
	*preprompt = ""
	ti.Focus()

	return model{
		textarea: ti,
		err:      nil,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case "ctrl+c":
			*loop = false
			*userInput = ""
			return m, tea.Quit

		case "ctrl+d":
			*userInput = m.textarea.Value()

			if len(*userInput) > 1 {
				m.textarea.Blur()
				return m, tea.Quit
			}
		case "ctrl+z":
			if runtime.GOOS != "windows" {
				return m, tea.Suspend
			}
		case "tab":
			if m.textarea.Focused() {
				m.textarea.InsertString("\t")
			}
		default:
			if m.textarea.Focused() {
				m.textarea, cmd = m.textarea.Update(msg)
				m.textarea.SetHeight(min(20, max(6, m.textarea.LineCount()+1)))
				cmds = append(cmds, cmd)
			}
		}

		// Command mode
		if !m.textarea.Focused() {
			switch msg.String() {
			case "i":
				m.textarea.Focus()
			case "c":
				if len(*lastResp) == 0 {
					break
				}
				err := clipboard.WriteAll(*lastResp)
				if err != nil {
					fmt.Println("Could not write to clipboard")
				}
			case "b":
				if len(*lastResp) == 0 {
					break
				}
				lastCodeBlock := utils.GetLastCodeBlock(*lastResp)
				err := clipboard.WriteAll(lastCodeBlock)
				if err != nil {
					fmt.Println("Could not write to clipboard")
				}
			case "p":
				m.textarea.Focus()
				clip, err := clipboard.ReadAll()
				if err != nil {
					fmt.Println("Could not read from clipboard")
					break
				}
				m.textarea.InsertString(clip)
				m.textarea.SetHeight(min(20, max(6, m.textarea.LineCount()+1)))
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() tea.View {
	return tea.NewView(m.textarea.View())
}

func GetFormattedInputStdin() (formattedInput string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()
	return strings.TrimSpace(input)
}

type InputModel struct {
	textinput textinput.Model
	history   []string
	histIdx   int
	tempInput string
	Value     string
	Canceled  bool
}

func InitialInputModel(prompt string, history []string) InputModel {
	ti := textinput.New()
	ti.Prompt = prompt
	ti.Focus()

	return InputModel{
		textinput: ti,
		history:   history,
		histIdx:   len(history),
		Value:     "",
		Canceled:  false,
	}
}

func (m InputModel) Init() tea.Cmd {
	return textinput.Blink
}

func (m InputModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.Canceled = true
			return m, tea.Quit

		case "ctrl+d":
			if m.textinput.Value() == "" {
				m.Canceled = true
				return m, tea.Quit
			}

		case "ctrl+z":
			if runtime.GOOS != "windows" {
				return m, tea.Suspend
			}

		case "enter":
			m.Value = m.textinput.Value()
			return m, tea.Quit

		case "up":
			if len(m.history) > 0 && m.histIdx > 0 {
				if m.histIdx == len(m.history) {
					m.tempInput = m.textinput.Value()
				}
				m.histIdx--
				m.textinput.SetValue(m.history[m.histIdx])
				m.textinput.CursorEnd()
			}
			return m, nil

		case "down":
			if m.histIdx < len(m.history) {
				m.histIdx++
				if m.histIdx == len(m.history) {
					m.textinput.SetValue(m.tempInput)
				} else {
					m.textinput.SetValue(m.history[m.histIdx])
				}
				m.textinput.CursorEnd()
			}
			return m, nil
		}
	}

	m.textinput, cmd = m.textinput.Update(msg)
	return m, cmd
}

func (m InputModel) View() tea.View {
	v := tea.NewView(m.textinput.View())
	v.Cursor = m.textinput.Cursor()
	return v
}

// PromptInput runs a single-line input prompt using Bubble Tea.
// Returns input string, canceled bool (true if Ctrl+C was pressed), and error.
func PromptInput(prompt string, history []string) (string, bool, error) {
	m := InitialInputModel(prompt, history)
	p := tea.NewProgram(m)
	finalModel, err := p.Run()
	if err != nil {
		return "", false, err
	}
	im := finalModel.(InputModel)
	if !im.Canceled {
		fmt.Printf("%s%s\n", prompt, im.Value)
	}
	return im.Value, im.Canceled, nil
}
