package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strings"
	"syscall"

	"github.com/aandrew-me/tgpt/v2/src/bubbletea"
	"github.com/aandrew-me/tgpt/v2/src/helper"
	"github.com/aandrew-me/tgpt/v2/src/imagegen"
	"github.com/aandrew-me/tgpt/v2/src/mcp"
	"github.com/aandrew-me/tgpt/v2/src/providers"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/tools"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	tea "charm.land/bubbletea/v2"
	"github.com/fatih/color"
)

const localVersion = "2.13.0"

var bold = color.New(color.Bold)
var blue = color.New(color.FgBlue)

var programLoop = true

type toolsFlagValue struct {
	enabled   bool
	toolNames []string
}

func (f *toolsFlagValue) String() string {
	if !f.enabled {
		return "false"
	}
	if len(f.toolNames) == 0 {
		return "true"
	}
	return strings.Join(f.toolNames, ",")
}

func (f *toolsFlagValue) Set(s string) error {
	f.enabled = true
	if s == "true" || s == "1" || s == "" {
		return nil
	}
	if s == "false" || s == "0" {
		f.enabled = false
		return nil
	}
	parts := strings.Split(s, ",")
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			f.toolNames = append(f.toolNames, p)
		}
	}
	return nil
}

func (f *toolsFlagValue) IsBoolFlag() bool {
	return true
}

func restoreTerminal() {
	bubbletea.RestoreTerminal()
}

func loadConfig(configPath string) {
	explicitConfig := configPath != ""

	if configPath == "" {
		defaultPaths := []string{"config.conf"}
		if homeDir, err := os.UserHomeDir(); err == nil {
			defaultPaths = append(defaultPaths, filepath.Join(homeDir, ".config", "tgpt", "config.conf"))
		}
		for _, p := range defaultPaths {
			if _, err := os.Stat(p); err == nil {
				configPath = p
				break
			}
		}
	}

	if configPath == "" {
		return
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		if explicitConfig {
			fmt.Fprintf(os.Stderr, "Warning: could not read config file %q: %v\n", configPath, err)
		}
		return
	}

	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		// Only set if not already defined in the environment
		if os.Getenv(key) == "" {
			os.Setenv(key, val)
		}
	}
}

// shellSystemPrompt builds the system prompt for the interactive shell mode.
// Must be called after helper.SetShellAndOSVars().
func shellSystemPrompt(useAliases bool) string {
	extraContext := ""
	if useAliases {
		extraContext = "You have access to shell aliases, functions, and environment variables. "
	}
	return fmt.Sprintf(
		"You are a powerful terminal assistant. Answer the needs of the user. "+
			"You can execute commands in the command line if needed. Always wrap the command with the xml tag `<cmd>`. "+
			"Only output a command when you think the user wants to execute a command. Execute only one command in one response. "+
			"The shell environment you are in is %s. The operating system you are on is %s. "+
			extraContext+
			"Examples: "+
			"User: list the files in my home dir. "+
			"Assistant: Sure. I will list the files under your home dir. <cmd>ls ~</cmd>",
		helper.ShellName, helper.OperatingSystem,
	)
}

// runInteractiveShellMode handles both the shell and alias interactive modes.
func runInteractiveShellMode(
	params structs.Params,
	preprompt, logFile, initialInput string,
	shouldExecuteCommand, useAliases bool,
) {
	if useAliases {
		bold.Print("Interactive Shell mode with aliases started. Press Ctrl + C or type exit to quit.\n\n")
	} else {
		bold.Print("Interactive Shell mode started. Press Ctrl + C or type exit to quit.\n\n")
	}
	helper.SetShellAndOSVars()

	systemPrompt := shellSystemPrompt(useAliases)

	var previousMessages []any
	threadID := utils.RandomString(36)
	history := []string{}
	commandRegex := regexp.MustCompile(`<cmd>(.*?)</cmd>`)

	getAndPrintResponse := func(input string) string {
		input = strings.TrimSpace(input)
		if input == "" {
			return ""
		}
		if input == "exit" {
			bold.Println("Exiting...")
			restoreTerminal()
			os.Exit(0)
		}
		if len(logFile) > 0 {
			utils.LogToFile(input, "USER_QUERY", logFile)
		}
		// Use preprompt for first message
		if len(previousMessages) == 0 {
			input = preprompt + input
		}

		params.PrevMessages = previousMessages
		params.ThreadID = threadID
		params.SystemPrompt = systemPrompt

		responseObjects, responseTxt := helper.GetData(input, params, structs.ExtraOptions{IsInteractiveShell: true, IsNormal: true, AutoExec: shouldExecuteCommand})

		if len(logFile) > 0 {
			utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", logFile)
		}

		matches := commandRegex.FindStringSubmatch(responseTxt)
		if len(matches) > 1 {
			return strings.TrimSpace(matches[1])
		}

		previousMessages = append(previousMessages, responseObjects...)
		history = append(history, input)
		return ""
	}

	execCmd := func(cmd string) {
		if cmd == "" {
			return
		}
		var output string
		executed := false

		if shouldExecuteCommand {
			fmt.Println()
			output = helper.ExecuteCommandWithCapture(helper.ShellName, helper.ShellOptions, cmd, true, useAliases)
			executed = true
		} else {
			confirmed, err := bubbletea.ConfirmMenu(fmt.Sprintf("\nExecute shell command: `%s` ?", cmd), true)
			if errors.Is(err, bubbletea.ErrInterrupted) {
				handleExit()
			}
			if confirmed {
				output = helper.ExecuteCommandWithCapture(helper.ShellName, helper.ShellOptions, cmd, true, useAliases)
				executed = true
			}
		}

		// Add command execution to conversation context
		if !executed {
			previousMessages = append(previousMessages, structs.DefaultMessage{
				Role:    "user",
				Content: fmt.Sprintf("Declined to execute command: %s", cmd),
			})
			return
		}
		previousMessages = append(previousMessages, structs.DefaultMessage{
			Role:    "user",
			Content: fmt.Sprintf("Executed command: %s", cmd),
		})

		// Add command output to conversation context only if it's not empty
		if output != "" {
			outputMsg := structs.DefaultMessage{
				Role:    "user",
				Content: fmt.Sprintf("Command output:\n%s", output),
			}
			previousMessages = append(previousMessages, outputMsg)
		}
	}

	input := strings.TrimSpace(initialInput)
	if input != "" {
		blue.Println("╭─ You")
		blue.Print("╰─> ")
		fmt.Println(input)
		cmd := getAndPrintResponse(input)
		execCmd(cmd)
	}

	for {
		blue.Println("╭─ You")
		input, canceled, err := bubbletea.PromptInput(blue.Sprint("╰─> "), history)
		if err != nil {
			utils.PrintError(err.Error())
			os.Exit(1)
		}
		if canceled {
			handleExit()
		}
		cmd := getAndPrintResponse(input)
		execCmd(cmd)
	}
}

func main() {
	var userInput string
	var lastResponse string
	var executablePath string

	// Parse --config manually before flag.Parse so config values are available
	// as defaults for other flags.
	var configPath string
	for i := 0; i < len(os.Args); i++ {
		arg := os.Args[i]
		if arg == "--config" || arg == "-config" {
			if i+1 < len(os.Args) {
				configPath = os.Args[i+1]
			}
		} else if val, ok := strings.CutPrefix(arg, "--config="); ok {
			configPath = val
		} else if val, ok := strings.CutPrefix(arg, "-config="); ok {
			configPath = val
		}
	}

	loadConfig(configPath)

	if execPath, err := os.Executable(); err == nil {
		executablePath = execPath
	}

	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-terminate
		restoreTerminal()
		os.Exit(130)
	}()

	apiModel := flag.String("model", "", "Choose which model to use")
	provider := flag.String("provider", "", "Choose which provider to use")
	apiKey := flag.String("key", os.Getenv("AI_API_KEY"), "Use personal API Key")
	temperature := flag.String("temperature", os.Getenv("TGPT_TEMPERATURE"), "Set temperature")
	top_p := flag.String("top_p", os.Getenv("TGPT_TOP_P"), "Set top_p")
	preprompt := flag.String("preprompt", "", "Set preprompt")
	flag.String("config", "", "Path to the configuration file")

	out := flag.String("out", "", "Output file path")
	width := flag.Int("width", 1024, "Output image width")
	height := flag.Int("height", 1024, "Output image height")

	imgNegative := flag.String("img_negative", "", "Negative prompt. Avoid generating specific elements or characteristics")
	imgCount := flag.String("img_count", "1", "Number of images you want to generate")
	imgRatio := flag.String("img_ratio", "1:1", "Image Aspect Ratio")

	url := flag.String("url", "", "url for openai providers")

	logFile := flag.String("log", "", "Filepath to log conversation to.")
	rotateProviders := flag.String("rotate", "", "Comma-separated fallback providers (Env: AI_ROTATE_PROVIDERS)")
	searchProvider := flag.String("search-provider", "", "Search provider: exa or google (Env: SEARCH_PROVIDER)")
	shouldExecuteCommand := flag.Bool("y", false, "Instantly execute the shell command")

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

	isVersion := flag.Bool("v", false, "Get version of tgpt")
	flag.BoolVar(isVersion, "version", false, "Get version of tgpt")

	isHelp := flag.Bool("h", false, "Show help message")
	flag.BoolVar(isHelp, "help", false, "Show help message")

	isUpdate := flag.Bool("u", false, "Update program")
	flag.BoolVar(isUpdate, "update", false, "Update program")

	isChangelog := flag.Bool("cl", false, "See changelog of versions")
	flag.BoolVar(isChangelog, "changelog", false, "See changelog of versions")

	mcpEnabled := flag.Bool("mcp", false, "Enable MCP support and auto-detect config file")
	mcpConfig := flag.String("mcp-config", os.Getenv("MCP_CONFIG"), "Path to MCP server configuration JSON file")
	mcpServer := flag.String("mcp-server", "", "Command to run a stdio MCP server directly")
	mcpAdd := flag.Bool("mcp-add", false, "Interactively add a new MCP server to mcp_config.json")
	mcpRemove := flag.Bool("mcp-remove", false, "Interactively remove an MCP server from mcp_config.json")
	var toolsFlag toolsFlagValue
	flag.Var(&toolsFlag, "t", "Enable tools / MCP support")
	flag.Var(&toolsFlag, "tools", "Enable tools / MCP support")

	isVerbose := flag.Bool("vb", false, "Enable verbose output for debugging")
	flag.BoolVar(isVerbose, "verbose", false, "Enable verbose output for debugging")

	flag.Parse()

	promptArgIndex := 0
	if toolsFlag.enabled && len(toolsFlag.toolNames) == 0 && flag.NArg() > 0 {
		if parsedTools, ok := tools.ParseToolList(flag.Arg(0)); ok {
			toolsFlag.toolNames = parsedTools
			promptArgIndex = 1
		}
	}

	if *mcpAdd {
		if err := mcp.AddServerInteractive(context.Background(), *mcpConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	if *mcpRemove {
		if err := mcp.RemoveServerInteractive(context.Background(), *mcpConfig); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		os.Exit(0)
	}

	var rotateProvidersSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "rotate" {
			rotateProvidersSet = true
		}
	})

	finalProvider := *provider
	if finalProvider == "" {
		if *isImage {
			finalProvider = os.Getenv("IMG_PROVIDER")
		} else {
			finalProvider = os.Getenv("AI_PROVIDER")
		}
	}

	rotateStr := *rotateProviders
	if !rotateProvidersSet && rotateStr == "" {
		rotateStr = os.Getenv("AI_ROTATE_PROVIDERS")
	}

	finalSearchProvider := *searchProvider
	if finalSearchProvider == "" {
		finalSearchProvider = os.Getenv("SEARCH_PROVIDER")
	}
	if finalSearchProvider == "" {
		finalSearchProvider = "exa"
	}

	var mcpConfigSet bool
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "mcp-config" {
			mcpConfigSet = true
		}
	})

	var activeTools []any
	mcpMgr := mcp.NewManager(tools.DefaultRegistry)
	defer mcpMgr.Close()

	mcpRequested := *mcpEnabled || mcpConfigSet || *mcpConfig != "" || *mcpServer != ""
	if toolsFlag.enabled || mcpRequested {
		if !providers.SupportsTools(finalProvider) {
			pName := finalProvider
			if pName == "" {
				pName = "opencode"
			}
			fmt.Fprintf(os.Stderr, "Warning: provider %q does not support tools or MCP. Tools will be ignored.\n", pName)
		} else {
			if toolsFlag.enabled {
				tools.DefaultRegistry.RegisterBuiltinTools(toolsFlag.toolNames...)
			}

			if mcpRequested {
				ctx := context.Background()
				cfg, err := mcp.LoadConfig(*mcpConfig)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Warning: failed to load MCP config: %v\n", err)
				} else if cfg != nil {
					for name, sc := range cfg.MCPServers {
						if err := mcpMgr.InitServer(ctx, name, sc); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to init MCP server %s: %v\n", name, err)
						}
					}
				} else if *mcpEnabled && *mcpServer == "" {
					fmt.Fprintf(os.Stderr, "Warning: no MCP config file found (checked mcp_config.json and ~/.config/tgpt/mcp_config.json)\n")
				}
				if *mcpServer != "" {
					parts := strings.Fields(*mcpServer)
					if len(parts) > 0 {
						sc := mcp.ServerConfig{Command: parts[0], Args: parts[1:]}
						if err := mcpMgr.InitServer(ctx, "cli-mcp", sc); err != nil {
							fmt.Fprintf(os.Stderr, "Warning: failed to init MCP server %s: %v\n", *mcpServer, err)
						}
					}
				}
			}

			activeTools = tools.DefaultRegistry.GetOpenAITools()
		}
	}

	mainParams := structs.Params{
		ApiKey:          *apiKey,
		ApiModel:        *apiModel,
		Provider:        finalProvider,
		Temperature:     *temperature,
		Top_p:           *top_p,
		Preprompt:       *preprompt,
		ThreadID:        "",
		Url:             *url,
		PrevMessages:    []any{},
		RotateProviders: rotateStr,
		Tools:           activeTools,
	}

	imageParams := structs.ImageParams{
		ImgRatio:          *imgRatio,
		ImgNegativePrompt: *imgNegative,
		ImgCount:          *imgCount,
		Width:             *width,
		Height:            *height,
		Out:               *out,
		Params:            mainParams,
	}

	prompt := flag.Arg(promptArgIndex)

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

	if len(os.Args) > 1 {
		switch {
		case *isVersion:
			fmt.Println("tgpt", localVersion)
		case *isChangelog:
			helper.GetVersionHistory()
		case *isImage:
			if len(prompt) > 0 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if trimmedPrompt == "" {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -img "cat"`)
					return
				}
				imagegen.GenerateImg(trimmedPrompt, imageParams, *isQuiet)
			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				if !*isQuiet {
					fmt.Println()
				}
				imagegen.GenerateImg(formattedInput, imageParams, *isQuiet)
			}

		case *isWhole:
			if len(prompt) > 0 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if trimmedPrompt == "" {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -w "What is encryption?"`)
					return
				}
				helper.GetWholeText(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					structs.ExtraOptions{IsGetWhole: *isWhole, AutoExec: *shouldExecuteCommand},
					mainParams,
				)
			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				helper.GetWholeText(
					*preprompt+formattedInput+cleanPipedInput,
					structs.ExtraOptions{IsGetWhole: *isWhole, AutoExec: *shouldExecuteCommand},
					mainParams,
				)
			}

		case *isShell:
			if len(prompt) > 0 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if trimmedPrompt == "" {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -s "How to update system"`)
					return
				}
				helper.ShellCommand(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					mainParams,
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
			if len(prompt) > 0 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if trimmedPrompt == "" {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -c "Hello world in Python"`)
					os.Exit(1)
				}
				helper.CodeGenerate(
					*preprompt+trimmedPrompt+contextText+pipedInput,
					mainParams,
					structs.ExtraOptions{
						IsGetCode:   true,
						IsGetSilent: *isQuiet,
						AutoExec:    *shouldExecuteCommand,
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
				if input == "" {
					return
				}
				if input == "exit" {
					bold.Println("Exiting...")
					restoreTerminal()
					os.Exit(0)
				}
				if len(*logFile) > 0 {
					utils.LogToFile(input, "USER_QUERY", *logFile)
				}
				// Use preprompt for first message
				if len(previousMessages) == 0 {
					input = *preprompt + input
				}

				mainParams.PrevMessages = previousMessages
				mainParams.ThreadID = threadID

				responseObjects, responseTxt := helper.GetData(input, mainParams, structs.ExtraOptions{IsInteractive: true, IsNormal: true, IsGetSilent: *isQuiet, AutoExec: *shouldExecuteCommand})

				if len(*logFile) > 0 {
					utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
				}

				previousMessages = append(previousMessages, responseObjects...)
				history = append(history, input)
				lastResponse = responseTxt
			}

			input := strings.TrimSpace(prompt)
			if input != "" {
				blue.Println("╭─ You")
				blue.Print("╰─> ")
				fmt.Println(input)
				getAndPrintResponse(input)
			}

			for {
				blue.Println("╭─ You")
				input, canceled, err := bubbletea.PromptInput(blue.Sprint("╰─> "), history)
				if err != nil {
					utils.PrintError(err.Error())
					os.Exit(1)
				}
				if canceled {
					handleExit()
				}
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

					mainParams.PrevMessages = previousMessages
					mainParams.ThreadID = threadID

					responseObjects, responseTxt := helper.GetData(userInput, mainParams, structs.ExtraOptions{IsInteractive: true, IsNormal: true, IsGetSilent: *isQuiet, AutoExec: *shouldExecuteCommand})
					previousMessages = append(previousMessages, responseObjects...)
					lastResponse = responseTxt

					if len(*logFile) > 0 {
						utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
					}
				}
			}

		case *isInteractiveShell:
			runInteractiveShellMode(mainParams, *preprompt, *logFile, prompt, *shouldExecuteCommand, false)

		case *isFind:
			/////////////////////
			// Find - One-shot web search
			/////////////////////

			if len(prompt) > 0 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if trimmedPrompt == "" {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -f "What is the latest news about AI?"`)
					return
				}

				// Validate search provider
				supportedSearchProviders := map[string]bool{"google": true, "exa": true, "": true}
				if !supportedSearchProviders[finalSearchProvider] {
					log.Fatal("Search provider is not valid")
				}

				extraOptions := structs.ExtraOptions{
					IsFind:         true,
					Verbose:        *isVerbose,
					SearchProvider: finalSearchProvider,
					AutoExec:       *shouldExecuteCommand,
				}

				helper.SearchQuery(trimmedPrompt, mainParams, extraOptions, *isQuiet, *logFile)
			} else {
				utils.PrintError("You need to provide some text")
				utils.PrintError(`Example: tgpt -f "What is the latest news about AI?"`)
			}

		case *isInteractiveFind:
			/////////////////////
			// Interactive Find - Interactive web search session
			/////////////////////

			bold.Print("Interactive Find mode started. Press Ctrl + C or type exit to quit.\n\n")

			extraOptions := structs.ExtraOptions{
				IsInteractiveFind: true,
				IsFind:            true,
				Verbose:           *isVerbose,
				SearchProvider:    finalSearchProvider,
				AutoExec:          *shouldExecuteCommand,
			}

			getAndPrintFindResponse := helper.InteractiveFindSession(mainParams, extraOptions, *logFile, nil)
			history := []string{}

			input := strings.TrimSpace(prompt)
			if input != "" {
				blue.Println("╭─ You")
				blue.Print("╰─> ")
				fmt.Println(input)
				getAndPrintFindResponse(input)
			}

			for {
				blue.Println("╭─ You")
				input, canceled, err := bubbletea.PromptInput(blue.Sprint("╰─> "), history)
				if err != nil {
					utils.PrintError(err.Error())
					os.Exit(1)
				}
				if canceled {
					handleExit()
				}
				if len(input) > 0 {
					getAndPrintFindResponse(input)
					history = append(history, input)
				}
			}

		case *isInteractiveAlias:
			runInteractiveShellMode(mainParams, *preprompt, *logFile, prompt, *shouldExecuteCommand, true)

		case *isHelp:
			helper.ShowHelpMessage()

		case *isQuiet:
			if len(prompt) > 0 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if trimmedPrompt == "" {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -q "What is encryption?"`)
					return
				}
				if _, _, err := helper.MakeRequestAndGetData(*preprompt+trimmedPrompt+contextText+pipedInput, mainParams, structs.ExtraOptions{IsGetSilent: true, AutoExec: *shouldExecuteCommand}); err != nil {
					return
				}
			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				fmt.Println()
				if _, _, err := helper.MakeRequestAndGetData(*preprompt+formattedInput+cleanPipedInput, mainParams, structs.ExtraOptions{IsGetSilent: true, AutoExec: *shouldExecuteCommand}); err != nil {
					return
				}
			}

		default:
			formattedInput := strings.TrimSpace(prompt)

			if formattedInput == "" {
				utils.PrintError("You need to write something")
				return
			}

			helper.GetData(
				*preprompt+formattedInput+contextText+pipedInput,
				mainParams,
				structs.ExtraOptions{
					IsNormal: true, IsInteractive: false, Verbose: *isVerbose, AutoExec: *shouldExecuteCommand,
				})
		}
	} else {
		scanner := bufio.NewScanner(os.Stdin)
		scanner.Scan()
		if err := scanner.Err(); err != nil {
			utils.PrintError(fmt.Sprintf("Error reading standard input: %v", err))
			return
		}
		input := scanner.Text()
		formattedInput := strings.TrimSpace(input)
		helper.GetData(*preprompt+formattedInput+pipedInput, mainParams, structs.ExtraOptions{IsInteractive: false, IsNormal: true, Verbose: *isVerbose, AutoExec: *shouldExecuteCommand})
	}
}

func handleExit() {
	bold.Println("Exiting...")
	restoreTerminal()
	os.Exit(130)
}

