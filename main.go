package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/aandrew-me/tgpt/v2/utils"
	"github.com/atotto/clipboard"
	Prompt "github.com/c-bata/go-prompt"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
	"github.com/olekukonko/ts"
)

const localVersion = "2.6.0"

var bold = color.New(color.Bold)
var boldBlue = color.New(color.Bold, color.FgBlue)
var blue = color.New(color.FgBlue)
var boldViolet = color.New(color.Bold, color.FgMagenta)
var codeText = color.New(color.BgBlack, color.FgGreen, color.Bold)
var stopSpin = false
var programLoop = true
var userInput = ""
var lastResponse = ""
var executablePath = ""
var provider *string
var apiModel *string
var apiKey *string
var temperature *string
var top_p *string
var max_length *string
var preprompt *string

func main() {
	execPath, err := os.Executable()
	if err == nil {
		executablePath = execPath
	}
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-terminate
		os.Exit(0)
	}()

	args := os.Args

	apiModel = flag.String("model", "", "Choose which provider to use")
	provider = flag.String("provider", "", "Choose which provider to use")
	apiKey = flag.String("key", "", "Use personal API Key")
	temperature = flag.String("temperature", "", "Set temperature")
	top_p = flag.String("top_p", "", "Set top_p")
	max_length = flag.String("max_length", "", "Set max length of response")
	preprompt = flag.String("preprompt", "", "Set preprompt")

	isQuiet := flag.Bool("q", false, "Gives response back without loading animation")
	flag.BoolVar(isQuiet, "quite", false, "Gives response back without loading animation")

	isWhole := flag.Bool("w", false, "Gives response back as a whole text")
	flag.BoolVar(isWhole, "whole", false, "Gives response back as a whole text")

	isCode := flag.Bool("c", false, "Generate Code. (Experimental)")
	flag.BoolVar(isCode, "code", false, "Generate Code. (Experimental)")

	isShell := flag.Bool("s", false, "Generate and Execute shell commands.")
	flag.BoolVar(isShell, "shell", false, "Generate and Execute shell commands.")

	isImage := flag.Bool("img", false, "Generate images from text")
	flag.BoolVar(isImage, "image", false, "Generate images from text")

	isInteractive := flag.Bool("i", false, "Start normal interactive mode")
	flag.BoolVar(isInteractive, "interactive", false, "Start normal interactive mode")

	isMultiline := flag.Bool("m", false, "Start multi-line interactive mode")
	flag.BoolVar(isMultiline, "multiline", false, "Start multi-line interactive mode")

	isVersion := flag.Bool("v", false, "Gives response back as a whole text")
	flag.BoolVar(isVersion, "version", false, "Gives response back as a whole text")

	isHelp := flag.Bool("h", false, "Gives response back as a whole text")
	flag.BoolVar(isHelp, "help", false, "Gives response back as a whole text")

	isUpdate := flag.Bool("u", false, "Update program")
	flag.BoolVar(isUpdate, "update", false, "Update program")

	isChangelog := flag.Bool("cl", false, "See changelog of versions")
	flag.BoolVar(isChangelog, "changelog", false, "See changelog of versions")

	flag.Parse()

	prompt := flag.Arg(0)

	pipedInput := ""
	cleanPipedInput := ""

	if len(args) > 1 {
		stat, _ := os.Stdin.Stat()

		// Checking for piped text
		if (stat.Mode() & os.ModeCharDevice) == 0 {
			scanner := bufio.NewScanner(os.Stdin)
			for scanner.Scan() {
				pipedInput += scanner.Text()
			}

			if err := scanner.Err(); err != nil {
				fmt.Fprintln(os.Stderr, "reading standard input:", err)
			}
		}

		if len(pipedInput) > 0 {
			cleanPipedInputByte, _ := json.Marshal(pipedInput)
			cleanPipedInput = string(cleanPipedInputByte)
			cleanPipedInput = cleanPipedInput[1 : len(cleanPipedInput)-1]

			safePipedBytes, _ := json.Marshal("\n\nHere is some text for the context:\n" + pipedInput + "\n")
			pipedInput = string(safePipedBytes)
			pipedInput = pipedInput[1 : len(pipedInput)-1]
		}

		if *isVersion {
			fmt.Println("tgpt", localVersion)
		} else if *isChangelog {
			getVersionHistory()
		} else if *isWhole {
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -w "What is encryption?"`)
					os.Exit(1)
				}
				getWholeText(trimmedPrompt + pipedInput)
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				getWholeText(formattedInput + cleanPipedInput)
			}
		} else if *isQuiet {
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -q "What is encryption?"`)
					os.Exit(1)
				}
				getSilentText(trimmedPrompt + pipedInput)
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				getSilentText(formattedInput + cleanPipedInput)
			}
		} else if *isShell {
			if len(prompt) > 1 {
				go loading(&stopSpin)
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -s "How to update system"`)
					os.Exit(1)
				}
				shellCommand(trimmedPrompt + pipedInput)
			} else {
				fmt.Fprintln(os.Stderr, "You need to provide some text")
				fmt.Fprintln(os.Stderr, `Example: tgpt -s "How to update system"`)
				os.Exit(1)
			}

		} else if *isCode {
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -c "Hello world in Python"`)
					os.Exit(1)
				}
				codeGenerate(trimmedPrompt + pipedInput)
			} else {
				fmt.Fprintln(os.Stderr, "You need to provide some text")
				fmt.Fprintln(os.Stderr, `Example: tgpt -c "Hello world in Python"`)
				os.Exit(1)
			}
		} else if *isUpdate {
			update()
		} else if *isInteractive {
			/////////////////////
			// Normal interactive
			/////////////////////

			bold.Print("Interactive mode started. Press Ctrl + C or type exit to quit.\n\n")

			previousMessages := ""
			threadID := utils.RandomString(36)
			history := []string{}

			for {
				blue.Println("╭─ You")
				input := Prompt.Input("╰─> ", historyCompleter,
					Prompt.OptionHistory(history),
					Prompt.OptionPrefixTextColor(Prompt.Blue),
					Prompt.OptionAddKeyBind(Prompt.KeyBind{
						Key: Prompt.ControlC,
						Fn:  exit,
					}),
				)

				if len(input) > 1 {
					input = strings.TrimSpace(input)
					if len(input) > 1 {
						if input == "exit" {
							bold.Println("Exiting...")
							if runtime.GOOS != "windows" {
								rawModeOff := exec.Command("stty", "-raw", "echo")
								rawModeOff.Stdin = os.Stdin
								_ = rawModeOff.Run()
								rawModeOff.Wait()
							}
							os.Exit(0)
						}
						responseJson, responseTxt := getData(input, true, structs.ExtraOptions{
							PrevMessages: previousMessages,
							ThreadID:     threadID,
							Provider:     *provider,
						})
						previousMessages += responseJson
						history = append(history, input)
						lastResponse = responseTxt
					}
				}
			}

		} else if *isMultiline {
			/////////////////////
			// Multiline interactive
			/////////////////////

			fmt.Print("\nPress Tab to submit, Ctrl + C to exit, Esc to unfocus, i to focus. When unfocused, press p to paste, c to copy response, bc to copy last code block in response\n")

			previousMessages := ""

			for programLoop {
				fmt.Print("\n")
				p := tea.NewProgram(initialModel())
				_, err := p.Run()

				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				if len(userInput) > 0 {
					responseJson, responseTxt := getData(userInput, true, structs.ExtraOptions{
						PrevMessages: previousMessages,
						Provider:     *provider,
					})
					previousMessages += responseJson
					lastResponse = responseTxt
				}

			}

		} else if *isImage {
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -img "cat"`)
					os.Exit(1)
				}
				generateImage(trimmedPrompt)
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				generateImage(formattedInput)
			}
		} else if *isHelp {
			showHelpMessage()
		} else {
			go loading(&stopSpin)
			formattedInput := strings.TrimSpace(prompt)

			if len(formattedInput) < 1 {
				fmt.Fprintln(os.Stderr, "You need to write something")
				os.Exit(1)
			}

			getData(formattedInput+pipedInput, false, structs.ExtraOptions{})
		}

	} else {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()
		go loading(&stopSpin)
		formattedInput := strings.TrimSpace(input)
		getData(formattedInput, false, structs.ExtraOptions{})
	}
}

// Multiline input
type errMsg error

type model struct {
	textarea textarea.Model
	err      error
}

func initialModel() model {
	size, _ := ts.GetSize()
	termWidth := size.Col()
	ti := textarea.New()
	ti.SetWidth(termWidth)
	ti.CharLimit = 200000
	ti.ShowLineNumbers = false
	ti.Placeholder = "Enter your prompt"
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
			programLoop = false
			userInput = ""
			return m, tea.Quit

		case tea.KeyTab:
			userInput = m.textarea.Value()

			if len(userInput) > 1 {
				m.textarea.Blur()
				return m, tea.Quit
			}
		default:
			if m.textarea.Focused() {
				m.textarea, cmd = m.textarea.Update(msg)
				cmds = append(cmds, cmd)
			}
		}

		// Command mode
		if !m.textarea.Focused() {
			switch msg.String() {
			case "i":
				m.textarea.Focus()
			case "c":
				if len(lastResponse) == 0 {
					break
				}
				err := clipboard.WriteAll(lastResponse)
				if err != nil {
					fmt.Println("Could not write to clipboard")
				}
			case "b":
				if len(lastResponse) == 0 {
					break
				}
				lastCodeBlock := getLastCodeBlock(lastResponse)
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
				userInput = clip
				m.textarea, cmd = m.textarea.Update(msg)
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

func getFormattedInputStdin() (formattedInput string) {
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	input := scanner.Text()
	return strings.TrimSpace(input)
}

func showHelpMessage() {
	boldBlue.Println(`Usage: tgpt [Flags] [Prompt]`)

	boldBlue.Println("\nFlags:")
	fmt.Printf("%-50v Generate and Execute shell commands. (Experimental) \n", "-s, --shell")
	fmt.Printf("%-50v Generate Code. (Experimental)\n", "-c, --code")
	fmt.Printf("%-50v Gives response back without loading animation\n", "-q, --quiet")
	fmt.Printf("%-50v Gives response back as a whole text\n", "-w, --whole")
	fmt.Printf("%-50v Generate images from text\n", "-img, --image")
	fmt.Printf("%-50v Set Provider. Detailed information has been provided below\n", "--provider")

	boldBlue.Println("\nSome additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.")
	fmt.Printf("%-50v Set Model\n", "--model")
	fmt.Printf("%-50v Set API Key\n", "--key")
	fmt.Printf("%-50v Set temperature\n", "--temperature")
	fmt.Printf("%-50v Set top_p\n", "--top_p")
	fmt.Printf("%-50v Set max response length\n", "--max_length")

	boldBlue.Println("\nOptions:")
	fmt.Printf("%-50v Print version \n", "-v, --version")
	fmt.Printf("%-50v Print help message \n", "-h, --help")
	fmt.Printf("%-50v Start normal interactive mode \n", "-i, --interactive")
	fmt.Printf("%-50v Start multi-line interactive mode \n", "-m, --multiline")
	fmt.Printf("%-50v See changelog of versions \n", "-cl, --changelog")

	if runtime.GOOS != "windows" {
		fmt.Printf("%-50v Update program \n", "-u, --update")
	}

	boldBlue.Println("\nProviders:")
	fmt.Println("The default provider is opengpts which uses 'GPT-3.5-turbo' model.")
	fmt.Println("Available providers to use: openai, opengpts, koboldai, phind, llama2")

	bold.Println("\nProvider: openai")
	fmt.Println("Needs API key to work and supports various models")

	bold.Println("\nProvider: opengpts")
	fmt.Println("Uses gpt-3.5-turbo only. Do not use with sensitive data")

	bold.Println("\nProvider: koboldai")
	fmt.Println("Uses koboldcpp/HF_SPACE_Tiefighter-13B only, answers from novels")

	bold.Println("\nProvider: phind")
	fmt.Println("Uses Phind Model. Great for developers")

	bold.Println("\nProvider: llama2")
	fmt.Println("Llama 2 is an open source large language model (LLM) developed by Meta AI. Uses llama2-70b by default. Supports other models.")

	boldBlue.Println("\nExamples:")
	fmt.Println(`tgpt "What is internet?"`)
	fmt.Println(`tgpt -m`)
	fmt.Println(`tgpt -s "How to update my system?"`)
	fmt.Println(`tgpt --provider opengpts "What is 1+1"`)
	fmt.Println(`tgpt --provider openai --key "sk-xxxx" --model "gpt-3.5-turbo" "What is 1+1"`)
	fmt.Println(`cat install.sh | tgpt "Explain the code"`)
}

func historyCompleter(d Prompt.Document) []Prompt.Suggest {
	s := []Prompt.Suggest{}
	return Prompt.FilterHasPrefix(s, d.GetWordAfterCursor(), true)
}

func exit(_ *Prompt.Buffer) {
	bold.Println("Exiting...")

	if runtime.GOOS != "windows" {
		rawModeOff := exec.Command("stty", "-raw", "echo")
		rawModeOff.Stdin = os.Stdin
		_ = rawModeOff.Run()
		rawModeOff.Wait()
	}

	os.Exit(0)
}
