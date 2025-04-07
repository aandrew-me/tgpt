package bubbletea

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/aandrew-me/tgpt/v2/src/helper"
	"github.com/atotto/clipboard"
	Prompt "github.com/c-bata/go-prompt"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/olekukonko/ts"
)

type model struct {
	textarea textarea.Model
	err      error
}

type errMsg error

var loop* bool
var userInput* string
var lastResp* string

func InitialModel(preprompt* string, programLoop* bool, lastResponse* string, userInputPtr* string) model {
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
		switch msg.Type {
		case tea.KeyEsc:
			if m.textarea.Focused() {
				m.textarea.Blur()
			}
		case tea.KeyCtrlC:
			*loop = false
			*userInput = ""
			return m, tea.Quit

		case tea.KeyCtrlD:
			*userInput = m.textarea.Value()

			if len(*userInput) > 1 {
				m.textarea.Blur()
				return m, tea.Quit
			}
		case tea.KeyTab:
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
				lastCodeBlock := helper.GetLastCodeBlock(*lastResp)
				err := clipboard.WriteAll(lastCodeBlock)
				if err != nil {
					fmt.Println("Could not write to clipboard")
				}
			case "p":
				m.textarea.Focus()
				clip, err := clipboard.ReadAll()
				msg.Runes = []rune(clip)
				if err != nil {
					fmt.Println("Could not read from clipboard")
				}
				*userInput = clip
				m.textarea, cmd = m.textarea.Update(msg)
				m.textarea.SetHeight(min(20, max(6, m.textarea.LineCount()+1)))
				cmds = append(cmds, cmd)
			}
		}

	// We handle errors just like any other message
	case errMsg:
		m.err = msg
		return m, nil
	}

	return m, tea.Batch(cmds...)
}

func (m model) View() string {
	return m.textarea.View()
}

func GetFormattedInputStdin() (formattedInput string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()
	return strings.TrimSpace(input)
}


func HistoryCompleter(d Prompt.Document) []Prompt.Suggest {
	s := []Prompt.Suggest{}
	return Prompt.FilterHasPrefix(s, d.GetWordAfterCursor(), true)
}