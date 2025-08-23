package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"syscall"

	"github.com/aandrew-me/tgpt/v2/src/bubbletea"
	"github.com/aandrew-me/tgpt/v2/src/config"
	"github.com/aandrew-me/tgpt/v2/src/helper"
	"github.com/aandrew-me/tgpt/v2/src/imagegen"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	Prompt "github.com/c-bata/go-prompt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

const localVersion = "2.11.0"

var bold = color.New(color.Bold)
var blue = color.New(color.FgBlue)

var programLoop = true

// getStringValue safely gets the string value from a *string, returning empty string if nil
func getStringValue(ptr *string) string {
	if ptr == nil {
		return ""
	}
	return *ptr
}

func main() {
	var userInput = ""
	var lastResponse = ""
	var executablePath = ""
	var provider *string
	var apiModel *string
	var apiKey *string
	var temperature *string
	var top_p *string
	var preprompt *string
	var url *string
	var logFile *string
	var shouldExecuteCommand *bool
	var out *string
	var height *int
	var width *int
	var imgNegative *string
	var imgCount *string
	var imgRatio *string

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
	provider = flag.String("provider", "", "Choose which provider to use")
	apiKey = flag.String("key", "", "Use personal API Key")
	temperature = flag.String("temperature", os.Getenv("TGPT_TEMPERATURE"), "Set temperature")
	top_p = flag.String("top_p", os.Getenv("TGPT_TOP_P"), "Set top_p")
	preprompt = flag.String("preprompt", "", "Set preprompt")

	out = flag.String("out", "", "Output file path")
	width = flag.Int("width", 1024, "Output image width")
	height = flag.Int("height", 1024, "Output image height")

	imgNegative = flag.String("img_negative", "", "Negative prompt. Avoid generating specific elements or characteristics")
	imgCount = flag.String("img_count", "1", "Number of images you want to generate")
	imgRatio = flag.String("img_ratio", "1:1", "Image Aspect Ratio")

	url = flag.String("url", "", "url for openai providers")

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

	isInteractiveShell := flag.Bool("is", false, "Start shell interactive mode")
	flag.BoolVar(isInteractiveShell, "interactive-shell", false, "Start shell interactive mode")

	isFind := flag.Bool("f", false, "Find information using web search")
	flag.BoolVar(isFind, "find", false, "Find information using web search")

	isInteractiveFind := flag.Bool("if", false, "Interactive find mode with web search")
	flag.BoolVar(isInteractiveFind, "interactive-find", false, "Interactive find mode with web search")

	isInteractiveAlias := flag.Bool("ia", false, "Start interactive shell mode with aliases and functions")
	flag.BoolVar(isInteractiveAlias, "interactive-alias", false, "Start interactive shell mode with aliases and functions")

	isVersion := flag.Bool("v", false, "Gives response back as a whole text")
	flag.BoolVar(isVersion, "version", false, "Gives response back as a whole text")

	isHelp := flag.Bool("h", false, "Gives response back as a whole text")
	flag.BoolVar(isHelp, "help", false, "Gives response back as a whole text")

	isUpdate := flag.Bool("u", false, "Update program")
	flag.BoolVar(isUpdate, "update", false, "Update program")

	isChangelog := flag.Bool("cl", false, "See changelog of versions")
	flag.BoolVar(isChangelog, "changelog", false, "See changelog of versions")

	isVerbose := flag.Bool("vb", false, "Enable verbose output for debugging")
	flag.BoolVar(isVerbose, "verbose", false, "Enable verbose output for debugging")

	profileName := flag.String("profile", "", "Use a configuration profile")

	// Add config CLI subcommand handling before flag parsing
	if len(os.Args) > 1 && os.Args[1] == "config" {
		helper.HandleConfigCommand(os.Args[2:])
		return
	}

	flag.Parse()

	// Load configuration system
	appConfig, err := config.LoadConfig("")
	if err != nil {
		// Non-critical error - continue with defaults if config fails to load
		if *isVerbose {
			fmt.Fprintf(os.Stderr, "Warning: Failed to load configuration: %v\n", err)
		}
		appConfig = config.GetDefaultConfig()
	}

	// Apply profile if specified
	if *profileName != "" {
		if profile, exists := appConfig.Profiles[*profileName]; exists {
			appConfig.ApplyProfile(profile)
		} else {
			utils.PrintError(fmt.Sprintf("Profile '%s' not found in configuration", *profileName))
			os.Exit(1)
		}
	}

	// Resolve configuration values using centralized precedence logic
	cliFlags := map[string]string{
		"provider":    getStringValue(provider),
		"key":         getStringValue(apiKey),
		"model":       getStringValue(apiModel),
		"temperature": getStringValue(temperature),
		"top_p":       getStringValue(top_p),
		"url":         getStringValue(url),
	}
	
	resolved := appConfig.ResolveConfig(cliFlags, *isImage)
	
	// Use resolved values
	final_provider := resolved.Provider
	effectiveApiKey := resolved.APIKey
	effectiveModel := resolved.Model
	effectiveTemperature := resolved.Temperature
	effectiveTopP := resolved.TopP
	effectiveUrl := resolved.URL

	main_params := structs.Params{
		ApiKey:       effectiveApiKey,
		ApiModel:     effectiveModel,
		Provider:     final_provider,
		Temperature:  effectiveTemperature,
		Top_p:        effectiveTopP,
		Preprompt:    *preprompt,
		ThreadID:     "",
		Url:          effectiveUrl,
		PrevMessages: []any{},
	}

	image_params := structs.ImageParams{
		ImgRatio:          *imgRatio,
		ImgNegativePrompt: *imgNegative,
		ImgCount:          *imgCount,
		Width:             *width,
		Height:            *height,
		Out:               *out,
		Params:            main_params,
	}

	prompt := flag.Arg(0)

	pipedInput := ""
	cleanPipedInput := ""
	contextText := ""

	stat, err := os.Stdin.Stat()

	if err != nil {
		utils.PrintError(fmt.Sprintf("Error accessing standard input: %v", err))

		return
	}

	// Checking for piped text
	if (stat.Mode() & os.ModeCharDevice) == 0 {
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			pipedInput += scanner.Text()
		}

		if err := scanner.Err(); err != nil {
			utils.PrintError(fmt.Sprintf("Error reading standard input: %v", err))

			return
		}
	}
	contextTextByte, _ := json.Marshal("\n\nHere is text for the context:\n")

	if len(pipedInput) > 0 {
		cleanPipedInputByte, err := json.Marshal(pipedInput)
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error marshaling piped input to JSON: %v", err))

			return
		}
		cleanPipedInput = string(cleanPipedInputByte)
		cleanPipedInput = cleanPipedInput[1 : len(cleanPipedInput)-1]

		safePipedBytes, err := json.Marshal(pipedInput + "\n")
		if err != nil {
			utils.PrintError(fmt.Sprintf("Error marshaling piped input to JSON: %v", err))

			return
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
			helper.GetVersionHistory()
		case *isImage:

			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -img "cat"`)

					return
				}

				imagegen.GenerateImg(trimmedPrompt, image_params, *isQuiet)

			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				if !*isQuiet {
					fmt.Println()
				}

				imagegen.GenerateImg(formattedInput, image_params, *isQuiet)
			}
		case *isWhole:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -w "What is encryption?"`)

					return
				}
				helper.GetWholeText(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					structs.ExtraOptions{IsGetWhole: *isWhole},
					main_params,
				)
			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				helper.GetWholeText(
					*preprompt+formattedInput+cleanPipedInput,
					structs.ExtraOptions{IsGetWhole: *isWhole},
					main_params,
				)
			}
		case *isShell:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -s "How to update system"`)

					return
				}
				helper.ShellCommand(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					main_params,
					structs.ExtraOptions{
						IsGetCommand: true,
						AutoExec:     *shouldExecuteCommand,
						IsGetSilent:  *isQuiet,
					},
				)
			} else {
				utils.PrintError("You need to provide some text")
				utils.PrintError(`Example: tgpt -s "How to update system"`)

				return
			}

		case *isCode:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -c "Hello world in Python"`)
					os.Exit(1)
				}
				helper.CodeGenerate(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					main_params,
					structs.ExtraOptions{
						IsGetCode:   true,
						IsGetSilent: *isQuiet,
					},
				)
			} else {
				utils.PrintError("You need to provide some text")
				utils.PrintError(`Example: tgpt -c "Hello world in Python"`)

				return
			}
		case *isUpdate:
			helper.Update(localVersion, executablePath)
		case *isInteractive:
			/////////////////////
			// Normal interactive
			/////////////////////

			bold.Print("Interactive mode started. Press Ctrl + C or type exit to quit.\n\n")

			var previousMessages []interface{}

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
				if len(previousMessages) == 0 {
					input = *preprompt + input
				}

				main_params.PrevMessages = append(main_params.PrevMessages, previousMessages...)
				main_params.ThreadID = threadID

				responseObjects, responseTxt := helper.GetData(input, main_params, structs.ExtraOptions{IsInteractive: true, IsNormal: true, IsGetSilent: *isQuiet})

				if len(*logFile) > 0 {
					utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
				}

				previousMessages = append(previousMessages, responseObjects...)
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
				input := Prompt.Input("╰─> ", bubbletea.HistoryCompleter,
					Prompt.OptionHistory(history),
					Prompt.OptionPrefixTextColor(Prompt.DarkBlue),
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

			var previousMessages []any

			threadID := utils.RandomString(36)

			for programLoop {
				fmt.Print("\n")
				p := tea.NewProgram(bubbletea.InitialModel(preprompt, &programLoop, &lastResponse, &userInput))
				_, err := p.Run()

				if err != nil {
					utils.PrintError(err.Error())

					os.Exit(1)
				}
				if len(userInput) > 0 {
					if len(*logFile) > 0 {
						utils.LogToFile(userInput, "USER_QUERY", *logFile)
					}

					main_params.PrevMessages = append(main_params.PrevMessages, previousMessages...)
					main_params.ThreadID = threadID

					responseObjects, responseTxt := helper.GetData(userInput, main_params, structs.ExtraOptions{IsInteractive: true, IsNormal: true, IsGetSilent: *isQuiet})
					previousMessages = append(previousMessages, responseObjects...)
					lastResponse = responseTxt

					if len(*logFile) > 0 {
						utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
					}
				}

			}

		case *isInteractiveShell:
			/////////////////////
			// shell interactive
			/////////////////////

			bold.Print("Interactive Shell mode started. Press Ctrl + C or type exit to quit.\n\n")
			helper.SetShellAndOSVars()
			promptIs := fmt.Sprintf("You are a powerful terminal assistant. Answer the needs of the user."+
				"You can execute command in command line if need. Always wrap the command with the xml tag `<cmd>`."+
				"Only output command when you think user wants to execute a command. Execute only one command in one response."+
				"The shell environment you are is %s. The operate system you are is %s."+
				"Examples:"+
				"User: list the files in my home dir."+
				"Assistant: Sure. I will list the files under your home dir. <cmd>ls ~</cmd>",
				helper.ShellName, helper.OperatingSystem,
			)
			var previousMessages []any

			threadID := utils.RandomString(36)
			history := []string{}

			getAndPrintResponse := func(input string) string {
				input = strings.TrimSpace(input)
				if len(input) <= 1 {
					return ""
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
				if len(previousMessages) == 0 {
					input = *preprompt + input
				}

				main_params.PrevMessages = previousMessages
				main_params.ThreadID = threadID
				main_params.SystemPrompt = promptIs

				responseObjects, responseTxt := helper.GetData(input, main_params, structs.ExtraOptions{IsInteractiveShell: true, IsNormal: true})
				// Regex to match complete <cmd>...</cmd>
				commandRegex := regexp.MustCompile(`<cmd>(.*?)</cmd>`)
				matches := commandRegex.FindStringSubmatch(responseTxt)
				if len(matches) > 1 {
					command := strings.TrimSpace(matches[1])
					// execute command
					return command
				}
				if len(*logFile) > 0 {
					utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
				}
				previousMessages = append(previousMessages, responseObjects...)
				history = append(history, input)
				lastResponse = responseTxt
				return ""
			}

			execCmd := func(cmd string) {
				if cmd != "" {
					var output string
					if *shouldExecuteCommand {
						fmt.Println()
						output = helper.ExecuteCommandWithCapture(helper.ShellName, helper.ShellOptions, cmd, true, false)
					} else {
						bold.Printf("\n\nExecute shell command: `%s` ? [y/n]: ", cmd)
						userInput := Prompt.Input("", bubbletea.HistoryCompleter,
							Prompt.OptionPrefixTextColor(Prompt.Blue),
							Prompt.OptionAddKeyBind(Prompt.KeyBind{
								Key: Prompt.ControlC,
								Fn:  exit,
							}),
						)
						userInput = strings.TrimSpace(userInput)

						if userInput == "y" || userInput == "" {
							output = helper.ExecuteCommandWithCapture(helper.ShellName, helper.ShellOptions, cmd, true, false)
						}
					}

					// Add command execution to conversation context
					commandMsg := structs.DefaultMessage{
						Role:    "user",
						Content: fmt.Sprintf("Executed command: %s", cmd),
					}
					previousMessages = append(previousMessages, commandMsg)

					// Add command output to conversation context only if it's not empty
					if output != "" {
						outputMsg := structs.DefaultMessage{
							Role:    "assistant",
							Content: fmt.Sprintf("Command output:\n%s", output),
						}
						previousMessages = append(previousMessages, outputMsg)
					}
				}
			}

			input := strings.TrimSpace(prompt)
			if len(input) > 1 {
				// if prompt is passed in interactive mode then send prompt as first message
				blue.Println("╭─ You")
				blue.Print("╰─> ")
				fmt.Println(input)
				cmd := getAndPrintResponse(input)
				execCmd(cmd)
			}

			for {
				blue.Println("╭─ You")
				input := Prompt.Input("╰─> ", bubbletea.HistoryCompleter,
					Prompt.OptionHistory(history),
					Prompt.OptionPrefixTextColor(Prompt.Blue),
					Prompt.OptionAddKeyBind(Prompt.KeyBind{
						Key: Prompt.ControlC,
						Fn:  exit,
					}),
				)
				cmd := getAndPrintResponse(input)
				execCmd(cmd)
			}

		case *isFind:
			/////////////////////
			// Find - One-shot web search
			/////////////////////

			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -f "What is the latest news about AI?"`)
					return
				}

				extraOptions := structs.ExtraOptions{
					IsFind:  true,
					Verbose: *isVerbose,
				}

				helper.SearchQuery(trimmedPrompt, main_params, extraOptions, *isQuiet, *logFile)
			} else {
				utils.PrintError("You need to provide some text")
				utils.PrintError(`Example: tgpt -f "What is the latest news about AI?"`)
			}

		case *isInteractiveFind:
			/////////////////////
			// Interactive Find - Interactive web search session
			/////////////////////

			bold.Print("Interactive Find mode started. Press Ctrl + C or type exit to quit.\n\n")

			// Set up interactive find session
			extraOptions := structs.ExtraOptions{
				IsInteractiveFind: true,
				IsFind:            true,
				Verbose:           *isVerbose,
			}

			helper.InteractiveFindSession(main_params, extraOptions, *logFile)

		case *isInteractiveAlias:
			/////////////////////
			// Interactive Alias - Interactive shell mode with aliases and functions
			/////////////////////

			bold.Print("Interactive Shell mode with aliases started. Press Ctrl + C or type exit to quit.\n\n")
			helper.SetShellAndOSVars()
			promptAlias := fmt.Sprintf("You are a powerful terminal assistant. Answer the needs of the user."+
				"You can execute command in command line if need. Always wrap the command with the xml tag `<cmd>`."+
				"Only output command when you think user wants to execute a command. Execute only one command in one response."+
				"The shell environment you are is %s. The operate system you are is %s."+
				"You have access to shell aliases, functions, and environment variables."+
				"Examples:"+
				"User: list the files in my home dir."+
				"Assistant: Sure. I will list the files under your home dir. <cmd>ls ~</cmd>",
				helper.ShellName, helper.OperatingSystem,
			)
			var previousMessages []any

			threadID := utils.RandomString(36)
			history := []string{}

			getAndPrintResponseAlias := func(input string) string {
				input = strings.TrimSpace(input)
				if len(input) <= 1 {
					return ""
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
				if len(previousMessages) == 0 {
					input = *preprompt + input
				}

				main_params.PrevMessages = previousMessages
				main_params.ThreadID = threadID
				main_params.SystemPrompt = promptAlias

				responseObjects, responseTxt := helper.GetData(input, main_params, structs.ExtraOptions{IsInteractiveShell: true, IsNormal: true})
				// Regex to match complete <cmd>...</cmd>
				commandRegex := regexp.MustCompile(`<cmd>(.*?)</cmd>`)
				matches := commandRegex.FindStringSubmatch(responseTxt)
				if len(matches) > 1 {
					command := strings.TrimSpace(matches[1])
					// execute command with aliases
					return command
				}
				if len(*logFile) > 0 {
					utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
				}
				previousMessages = append(previousMessages, responseObjects...)
				history = append(history, input)
				lastResponse = responseTxt
				return ""
			}

			execCmdAlias := func(cmd string) {
				if cmd != "" {
					var output string
					if *shouldExecuteCommand {
						fmt.Println()
						output = helper.ExecuteCommandWithCapture(helper.ShellName, helper.ShellOptions, cmd, true, true)
					} else {
						bold.Printf("\n\nExecute shell command: `%s` ? [y/n]: ", cmd)
						userInput := Prompt.Input("", bubbletea.HistoryCompleter,
							Prompt.OptionPrefixTextColor(Prompt.Blue),
							Prompt.OptionAddKeyBind(Prompt.KeyBind{
								Key: Prompt.ControlC,
								Fn:  exit,
							}),
						)
						userInput = strings.TrimSpace(userInput)

						if userInput == "y" || userInput == "" {
							output = helper.ExecuteCommandWithCapture(helper.ShellName, helper.ShellOptions, cmd, true, true)
						}
					}

					// Add command execution to conversation context
					commandMsg := structs.DefaultMessage{
						Role:    "user",
						Content: fmt.Sprintf("Executed command: %s", cmd),
					}
					previousMessages = append(previousMessages, commandMsg)

					// Add command output to conversation context only if it's not empty
					if output != "" {
						outputMsg := structs.DefaultMessage{
							Role:    "assistant",
							Content: fmt.Sprintf("Command output:\n%s", output),
						}
						previousMessages = append(previousMessages, outputMsg)
					}
				}
			}

			input := strings.TrimSpace(prompt)
			if len(input) > 1 {
				// if prompt is passed in interactive mode then send prompt as first message
				blue.Println("╭─ You")
				blue.Print("╰─> ")
				fmt.Println(input)
				cmd := getAndPrintResponseAlias(input)
				execCmdAlias(cmd)
			}

			for {
				blue.Println("╭─ You")
				input := Prompt.Input("╰─> ", bubbletea.HistoryCompleter,
					Prompt.OptionHistory(history),
					Prompt.OptionPrefixTextColor(Prompt.Blue),
					Prompt.OptionAddKeyBind(Prompt.KeyBind{
						Key: Prompt.ControlC,
						Fn:  exit,
					}),
				)
				cmd := getAndPrintResponseAlias(input)
				execCmdAlias(cmd)
			}

		case *isHelp:
			helper.ShowHelpMessage()
		case *isQuiet:
			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -q "What is encryption?"`)

					return
				}
				helper.MakeRequestAndGetData(*preprompt+trimmedPrompt+contextText+pipedInput, main_params, structs.ExtraOptions{IsGetSilent: true})
			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				fmt.Println()
				helper.MakeRequestAndGetData(*preprompt+formattedInput+cleanPipedInput, main_params, structs.ExtraOptions{IsGetSilent: true})
			}
		default:
			formattedInput := strings.TrimSpace(prompt)

			if len(formattedInput) <= 1 {
				utils.PrintError("You need to write something")

				return
			}

			helper.GetData(
				*preprompt+formattedInput+contextText+pipedInput,
				main_params,
				structs.ExtraOptions{
					IsNormal: true, IsInteractive: false,
				})
		}

	} else {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		input := scanner.Text()
		formattedInput := strings.TrimSpace(input)
		helper.GetData(*preprompt+formattedInput+pipedInput, main_params, structs.ExtraOptions{IsInteractive: false})
	}
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
