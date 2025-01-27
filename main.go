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

const localVersion = "2.8.3"

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
var url *string
var logFile *string
var shouldExecuteCommand *bool

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

	apiModel = flag.String("model", "", "Choose which model to use")
	provider = flag.String("provider", os.Getenv("AI_PROVIDER"), "Choose which provider to use")
	apiKey = flag.String("key", os.Getenv("AI_API_KEY"), "Use personal API Key")
	temperature = flag.String("temperature", "", "Set temperature")
	top_p = flag.String("top_p", "", "Set top_p")
	max_length = flag.String("max_length", "", "Set max length of response")
	preprompt = flag.String("preprompt", "", "Set preprompt")
	url = flag.String("url", "https://api.deepseek.com/v1/chat/completions", "url for deepseek providers")
	logFile = flag.String("log", "", "Filepath to log conversation to.")
	shouldExecuteCommand = flag.Bool(("y"), false, "Instantly execute the shell command")

	isQuiet := flag.Bool("q", false, "Gives response back without loading animation")
	flag.BoolVar(isQuiet, "quiet", false, "Gives response back without loading animation")

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
	contextText := ""

	stat, err := os.Stdin.Stat()
	if err != nil {
		fmt.Fprintln(os.Stderr, "accessing standard input:", err)
		os.Exit(1)
	}

	// Checking for piped text
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			pipedInput += scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			fmt.Fprintln(os.Stderr, "reading standard input:", err)
			os.Exit(1)
		}
	}
	contextTextByte, _ := json.Marshal("\n\nHere is text for the context:\n")

	if len(pipedInput) > 0 {
		cleanPipedInputByte, err := json.Marshal(pipedInput)
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshaling piped input to JSON:", err)
			os.Exit(1)
		}
		cleanPipedInput = string(cleanPipedInputByte)
		cleanPipedInput = cleanPipedInput[1 : len(cleanPipedInput)-1]

		safePipedBytes, err := json.Marshal(pipedInput + "\n")
		if err != nil {
			fmt.Fprintln(os.Stderr, "marshaling piped input to JSON:", err)
			os.Exit(1)
		}
		pipedInput = string(safePipedBytes)
		pipedInput = pipedInput[1 : len(pipedInput)-1]
		contextText = string(contextTextByte)
	}

	if len(*preprompt) > 0 {
		*preprompt += "\n"
	}

	if len(args) > 1 {
		switch {

		case *isVersion:
			fmt.Println("tgpt", localVersion)
		case *isChangelog:
			getVersionHistory()
		case *isWhole:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -w "What is encryption?"`)
					os.Exit(1)
				}
				getWholeText(*preprompt+trimmedPrompt+contextText+pipedInput, structs.ExtraOptions{IsGetWhole: *isWhole})
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				getWholeText(*preprompt+formattedInput+cleanPipedInput, structs.ExtraOptions{IsGetWhole: *isWhole})
			}
		case *isQuiet:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -q "What is encryption?"`)
					os.Exit(1)
				}
				getSilentText(*preprompt + trimmedPrompt + contextText + pipedInput, structs.ExtraOptions{})
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				getSilentText(*preprompt + formattedInput + cleanPipedInput, structs.ExtraOptions{})
			}
		case *isShell:
			if len(prompt) > 1 {
				go loading(&stopSpin)
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -s "How to update system"`)
					os.Exit(1)
				}
				shellCommand(*preprompt + trimmedPrompt + contextText + pipedInput)
			} else {
				fmt.Fprintln(os.Stderr, "You need to provide some text")
				fmt.Fprintln(os.Stderr, `Example: tgpt -s "How to update system"`)
				os.Exit(1)
			}

		case *isCode:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -c "Hello world in Python"`)
					os.Exit(1)
				}
				codeGenerate(*preprompt + trimmedPrompt + contextText + pipedInput)
			} else {
				fmt.Fprintln(os.Stderr, "You need to provide some text")
				fmt.Fprintln(os.Stderr, `Example: tgpt -c "Hello world in Python"`)
				os.Exit(1)
			}
		case *isUpdate:
			update()
		case *isInteractive:
			/////////////////////
			// Normal interactive
			/////////////////////

			bold.Print("Interactive mode started. Press Ctrl + C or type exit to quit.\n\n")

			previousMessages := ""
			threadID := utils.RandomString(36)
			history := []string{}

			getAndPrintResponse := func(input string) {
				input = strings.TrimSpace(input)
				if len(input) <= 1 {
					return
				}
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
				if len(*logFile) > 0 {
					utils.LogToFile(input, "USER_QUERY", *logFile)
				}
				// Use preprompt for first message
				if previousMessages == "" {
					input = *preprompt + input
				}
				responseJson, responseTxt := getData(input, structs.Params{
					PrevMessages: previousMessages,
					ThreadID:     threadID,
					Provider:     *provider,
				}, structs.ExtraOptions{IsInteractive: true, IsNormal: true})
				if len(*logFile) > 0 {
					utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
				}
				previousMessages += responseJson
				history = append(history, input)
				lastResponse = responseTxt

			}

			input := strings.TrimSpace(prompt)
			if len(input) > 1 {
				// if prompt is passed in interactive mode then send prompt as first message
				blue.Println("╭─ You")
				blue.Print("╰─> ")
				fmt.Println(input)
				getAndPrintResponse(input)
			}

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
				getAndPrintResponse(input)

			}

		case *isMultiline:
			/////////////////////
			// Multiline interactive
			/////////////////////

			fmt.Print("\nPress Ctrl + D to submit, Ctrl + C to exit, Esc to unfocus, i to focus. When unfocused, press p to paste, c to copy response, b to copy last code block in response\n")

			previousMessages := ""
			threadID := utils.RandomString(36)

			for programLoop {
				fmt.Print("\n")
				p := tea.NewProgram(initialModel())
				_, err := p.Run()

				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					os.Exit(1)
				}
				if len(userInput) > 0 {
					if len(*logFile) > 0 {
						utils.LogToFile(userInput, "USER_QUERY", *logFile)
					}

					responseJson, responseTxt := getData(userInput, structs.Params{
						PrevMessages: previousMessages,
						Provider:     *provider,
						ThreadID:     threadID,
					}, structs.ExtraOptions{IsInteractive: true, IsNormal: true})
					previousMessages += responseJson
					lastResponse = responseTxt

					if len(*logFile) > 0 {
						utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
					}
				}

			}

		case *isImage:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					fmt.Fprintln(os.Stderr, "You need to provide some text")
					fmt.Fprintln(os.Stderr, `Example: tgpt -img "cat"`)
					os.Exit(1)
				}
				generateImagePollinations(trimmedPrompt)
			} else {
				formattedInput := getFormattedInputStdin()
				fmt.Println()
				generateImagePollinations(*preprompt + formattedInput)
			}
		case *isHelp:
			showHelpMessage()
		default:
			go loading(&stopSpin)
			formattedInput := strings.TrimSpace(prompt)

			if len(formattedInput) < 1 {
				fmt.Fprintln(os.Stderr, "You need to write something")
				os.Exit(1)
			}

			getData(*preprompt+formattedInput+contextText+pipedInput, structs.Params{}, structs.ExtraOptions{IsNormal: true, IsInteractive: false, })
		}

	} else {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()
		go loading(&stopSpin)
		formattedInput := strings.TrimSpace(input)
		getData(*preprompt+formattedInput+pipedInput, structs.Params{}, structs.ExtraOptions{IsInteractive: false, })
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
			programLoop = false
			userInput = ""
			return m, tea.Quit

		case tea.KeyCtrlD:
			userInput = m.textarea.Value()

			if len(userInput) > 1 {
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
	fmt.Printf("%-50v Set Provider. Detailed information has been provided below. (Env: AI_PROVIDER)\n", "--provider")

	boldBlue.Println("\nSome additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.")
	fmt.Printf("%-50v Set Model\n", "--model")
	fmt.Printf("%-50v Set API Key. (Env: AI_API_KEY)\n", "--key")
	fmt.Printf("%-50v Set OpenAI API endpoint url\n", "--url")
	fmt.Printf("%-50v Set temperature\n", "--temperature")
	fmt.Printf("%-50v Set top_p\n", "--top_p")
	fmt.Printf("%-50v Set max response length\n", "--max_length")
	fmt.Printf("%-50v Set filepath to log conversation to (For interactive modes)\n", "--log")
	fmt.Printf("%-50v Set preprompt\n", "--preprompt")
	fmt.Printf("%-50v Execute shell command without confirmation\n", "-y")

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
	fmt.Println("The default provider is phind. The AI_PROVIDER environment variable can be used to specify a different provider.")
	fmt.Println("Available providers to use: blackboxai, duckduckgo, groq, isou, koboldai, ollama, openai and phind")

	bold.Println("\nProvider: blackboxai")
	fmt.Println("Uses BlackBox model. Great for developers")

	bold.Println("\nProvider: duckduckgo")
	fmt.Println("Available models: gpt-4o-mini (default), meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo, mistralai/Mixtral-8x7B-Instruct-v0.1, claude-3-haiku-20240307")

	bold.Println("\nProvider: groq")
	fmt.Println("Requires a free API Key. Supports LLaMA2-70b & Mixtral-8x7b")

	bold.Println("\nProvider: isou")
	fmt.Println("Supports DeepSeek API")

	bold.Println("\nProvider: koboldai")
	fmt.Println("Uses koboldcpp/HF_SPACE_Tiefighter-13B only, answers from novels")

	bold.Println("\nProvider: ollama")
	fmt.Println("Needs to be run locally. Supports many models")

	bold.Println("\nProvider: openai")
	fmt.Println("Needs API key to work and supports various models. Recognizes the OPENAI_API_KEY and OPENAI_MODEL environment variables. Supports custom urls with --url")

	bold.Println("\nProvider: phind")
	fmt.Println("Uses Phind Model. Great for developers")

	boldBlue.Println("\nExamples:")
	fmt.Println(`tgpt "What is internet?"`)
	fmt.Println(`tgpt -m`)
	fmt.Println(`tgpt -s "How to update my system?"`)
	fmt.Println(`tgpt --provider duckduckgo "What is 1+1"`)
	fmt.Println(`tgpt --provider openai --key "sk-xxxx" --model "gpt-3.5-turbo" "What is 1+1"`)
	fmt.Println(`tgpt --provider isou --key "sk-xxxx" --model "deepseek-chat" --url "https://api.deepseek.com/v1/chat/completions" "What is 1+1"`)
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
