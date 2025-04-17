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

	"github.com/aandrew-me/tgpt/v2/src/bubbletea"
	"github.com/aandrew-me/tgpt/v2/src/helper"
	"github.com/aandrew-me/tgpt/v2/src/imagegen"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	Prompt "github.com/c-bata/go-prompt"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/fatih/color"
)

const localVersion = "2.9.5"

var bold = color.New(color.Bold)
var blue = color.New(color.FgBlue)

var programLoop = true

func main() {
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
	var out *string
	var height *int
	var width *int

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
	out = flag.String("out", "", "Output file path")
	width = flag.Int("width", 1024, "Output image width")
	height = flag.Int("height", 1024, "Output image height")

	defaultUrl := ""
	if *provider == "openai" {
		// ideally default value should be inside openai provider file. To retain existing behavior and avoid braking change default value for openai is set here.
		defaultUrl = "https://api.openai.com/v1/chat/completions"
	}
	url = flag.String("url", defaultUrl, "url for openai providers")

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

	main_params := structs.Params{
		ApiKey: *apiKey,
		ApiModel: *apiModel,
		Provider: *provider,
		Temperature: *temperature,
		Top_p: *top_p,
		Max_length: *max_length,
		Preprompt: *preprompt,
		ThreadID: "",
		Url: *url,
		PrevMessages: "",
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
			params := structs.ImageParams{
				Params: main_params,
				Width:  *width,
				Height: *height,
				Out:    *out,
			}

			if len(prompt) > 1 {
				trimmedPrompt := strings.TrimSpace(prompt)
				if len(trimmedPrompt) < 1 {
					utils.PrintError("You need to provide some text")
					utils.PrintError(`Example: tgpt -img "cat"`)
					
					return
				}

				imagegen.GenerateImg(trimmedPrompt, params, *isQuiet)

			} else {
				formattedInput := bubbletea.GetFormattedInputStdin()
				if !*isQuiet {
					fmt.Println()
				}

				imagegen.GenerateImg(formattedInput, params, *isQuiet)
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
				helper.MakeRequestAndGetData(*preprompt+formattedInput+cleanPipedInput, main_params, structs.ExtraOptions{IsGetSilent: true},)
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
					*preprompt + trimmedPrompt + contextText + pipedInput,
					main_params,
					structs.ExtraOptions{
						IsGetCommand: true,
						AutoExec: *shouldExecuteCommand,
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
					*preprompt + trimmedPrompt + contextText + pipedInput,
					main_params,
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

				main_params.PrevMessages = previousMessages
				main_params.ThreadID = threadID

				responseJson, responseTxt := helper.GetData(input, main_params, structs.ExtraOptions{IsInteractive: true, IsNormal: true})
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
				input := Prompt.Input("╰─> ", bubbletea.HistoryCompleter,
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

					main_params.PrevMessages = previousMessages
					main_params.ThreadID = threadID

					responseJson, responseTxt := helper.GetData(userInput, main_params, structs.ExtraOptions{IsInteractive: true, IsNormal: true})
					previousMessages += responseJson
					lastResponse = responseTxt

					if len(*logFile) > 0 {
						utils.LogToFile(responseTxt, "ASSISTANT_RESPONSE", *logFile)
					}
				}

			}

		case *isHelp:
			helper.ShowHelpMessage()
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
