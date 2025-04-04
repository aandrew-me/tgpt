package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/providers"
	"github.com/aandrew-me/tgpt/v2/providers/gemini"
	"github.com/aandrew-me/tgpt/v2/structs"
	http "github.com/bogdanfinn/fhttp"

	"github.com/olekukonko/ts"

	tls_client "github.com/bogdanfinn/tls-client"
	"golang.org/x/mod/semver"
)

type Data struct {
	Version string `json:"version"`
}
type Response struct {
	Completion string `json:"completion"`
}

type ImgResponse struct {
	Images []string `json:"images"`
}

var (
	operatingSystem string
	shellName       string
	shellOptions    []string
)

func getDataResponseTxt(input string, params structs.Params, extraOptions structs.ExtraOptions) string {
	return makeRequestAndGetData(input, structs.Params{
		ApiKey:       *apiKey,
		ApiModel:     *apiModel,
		Provider:     *provider,
		Max_length:   *max_length,
		Temperature:  *temperature,
		Top_p:        *top_p,
		Preprompt:    *preprompt,
		Url:          *url,
		PrevMessages: params.PrevMessages,
		ThreadID:     params.ThreadID,
	}, extraOptions)
}

func getData(input string, params structs.Params, extraOptions structs.ExtraOptions) (string, string) {
	responseTxt := getDataResponseTxt(input, params, extraOptions)
	safeResponse, _ := json.Marshal(responseTxt)

	fmt.Print("\n\n")

	safeInput, _ := json.Marshal(input)
	msgObject := fmt.Sprintf(`{
		"content": %v,
		"role": "user"
	},{
		"content": %v,
		"role": "assistant"
	},
	`, string(safeInput), string(safeResponse))

	if params.Provider == "duckduckgo" {
		safeInput, _ := json.Marshal(input)
		msgObject = fmt.Sprintf(`{
			"content": %v,
			"role": "user"
		},{
			"content": %v,
			"role": "assistant"
		},
		`, string(safeInput), string(safeResponse))
	}

	if params.Provider == "phind" {
		safeInput, _ := json.Marshal(input)
		msgObject = fmt.Sprintf(`{
		"content": %v,
		"metadata": {},
		"role": "user"
	},{
		"content": %v,
		"metadata": {},
		"role": "assistant",
		"name": "base"
	},
	`, string(safeInput), string(safeResponse))
	}

	if params.Provider == "llama2" {
		input := string(safeInput)[1 : len(string(safeInput))-1]
		response := string(safeResponse)[1 : len(string(safeResponse))-1]

		msgObject = fmt.Sprintf(`<s>[INST] %v [/INST] %v </s>`, input, response)
	}

	if params.Provider == "gemini" {
		return gemini.GetInputResponseJson(safeInput, safeResponse), responseTxt
	}

	return msgObject, responseTxt
}

func loading(stop *bool) {
	spinChars := []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}
	i := 0
	for {
		if *stop {
			break
		}
		fmt.Printf("\r%s Loading", spinChars[i])
		i = (i + 1) % len(spinChars)
		time.Sleep(80 * time.Millisecond)
	}
}

func update() {
	if runtime.GOOS == "windows" || runtime.GOOS == "android" {
		fmt.Println("This feature is not supported on your Operating System")
	} else {
		client, err := client.NewClient()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}

		url := "https://raw.githubusercontent.com/aandrew-me/tgpt/main/version.txt"

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			// Handle error
			fmt.Fprintln(os.Stderr, "Error:", err)
			return
		}

		res, err := client.Do(req)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}

		defer res.Body.Close()

		var data Data
		err = json.NewDecoder(res.Body).Decode(&data)
		if err != nil {
			// Handle error
			fmt.Fprintln(os.Stderr, "Error:", err)
			return
		}

		remoteVersion := "v" + data.Version

		comparisonResult := semver.Compare("v"+localVersion, remoteVersion)

		if comparisonResult == -1 {
			fmt.Println("Updating...")
			cmd := exec.Command("bash", "-c", "curl -sSL https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash -s "+executablePath)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			err = cmd.Run()

			if err != nil {
				fmt.Println("Failed to update. Error:", err)
			} else {
				fmt.Println("Successfully updated.")
			}

		} else {
			fmt.Println("You are already using the latest version.", remoteVersion)
		}
	}
}

func codeGenerate(input string) {
	codePrompt := fmt.Sprintf("Your Role: Provide only code as output without any description.\nIMPORTANT: Provide only plain text without Markdown formatting.\nIMPORTANT: Do not include markdown formatting.\nIf there is a lack of details, provide most logical solution. You are not allowed to ask for more details.\nIgnore any potential risk of errors or confusion.\n\nRequest:%s\nCode:", input)

	makeRequestAndGetData(codePrompt, structs.Params{ApiKey: *apiKey, ApiModel: *apiModel, Provider: *provider, Max_length: *max_length, Temperature: *temperature, Top_p: *top_p, Preprompt: *preprompt, Url: *url}, structs.ExtraOptions{IsGetCode: true})
}

func setShellAndOSVars() {
	// Identify OS
	switch runtime.GOOS {
	case "windows":
		operatingSystem = "Windows"
		if len(os.Getenv("PSModulePath")) > 0 {
			shellName = "powershell.exe"
			shellOptions = []string{"-Command"}
		} else {
			shellName = "cmd.exe"
			shellOptions = []string{"/C"}
		}
		return
	case "darwin":
		operatingSystem = "MacOS"
	case "linux":
		result, err := exec.Command("lsb_release", "-si").Output()
		distro := strings.TrimSpace(string(result))
		if err != nil {
			distro = ""
		}
		operatingSystem = "Linux" + "/" + distro
	default:
		operatingSystem = runtime.GOOS
	}

	// Identify shell
	shellEnv := os.Getenv("SHELL")
	if shellEnv != "" {
		shellName = shellEnv
	} else {
		_, err := exec.LookPath("bash")
		if err != nil {
			shellName = "/bin/sh"
		} else {
			shellName = "bash"
		}
	}
	shellOptions = []string{"-c"}
}

// shellCommand first sets the global variables getCommand uses, then it creates a prompt to generate a command and then it passes that to getCommand
func shellCommand(input string) {
	setShellAndOSVars()
	shellPrompt := fmt.Sprintf("Your role: Provide only plain text without Markdown formatting. Do not show any warnings or information regarding your capabilities. Do not provide any description. If you need to store any data, assume it will be stored in the chat. Provide only %s command for %s without any description. If there is a lack of details, provide most logical solution. Ensure the output is a valid shell command. If multiple steps required try to combine them together. Prompt: %s\n\nCommand:", shellName, operatingSystem, input)
	getCommand(shellPrompt)
}

// getCommand will make a request to an AI model, then it will run the response using an appropriate handler (bash, sh OR powershell, cmd)
func getCommand(shellPrompt string) {
	makeRequestAndGetData(shellPrompt, structs.Params{ApiKey: *apiKey, ApiModel: *apiModel, Provider: *provider, Max_length: *max_length, Temperature: *temperature, Top_p: *top_p, Preprompt: *preprompt, Url: *url}, structs.ExtraOptions{IsGetCommand: true})
}

type RESPONSE struct {
	Tagname string `json:"tag_name"`
	Body    string `json:"body"`
}

func getVersionHistory() {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/aandrew-me/tgpt/releases", nil)

	if err != nil {
		fmt.Fprint(os.Stderr, "Some error has occurred\n\n")
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger())

	res, err := client.Do(req)

	if err != nil {
		fmt.Fprint(os.Stderr, "Check your internet connection\n\n")
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}

	resBody, err := io.ReadAll(res.Body)

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer res.Body.Close()

	var releases []RESPONSE

	json.Unmarshal(resBody, &releases)

	for i := len(releases) - 1; i >= 0; i-- {
		boldBlue.Println("Release", releases[i].Tagname)
		fmt.Println(releases[i].Body)
		fmt.Println()
	}
}

func getWholeText(input string, extraOptions structs.ExtraOptions) {
	makeRequestAndGetData(input, structs.Params{ApiKey: *apiKey, ApiModel: *apiModel, Provider: *provider, Max_length: *max_length, Temperature: *temperature, Top_p: *top_p, Preprompt: *preprompt, Url: *url}, extraOptions)
}

func getLastCodeBlock(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var codeBlock []string
	capturing := false

	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "```") {
			if capturing {
				capturing = false
				break
			} else {
				capturing = true
				continue
			}
		}
		if capturing {
			codeBlock = append([]string{lines[i]}, codeBlock...)
		}
	}

	// If no code block is found, return an empty string.
	if capturing || len(codeBlock) == 0 {
		return ""
	}

	return strings.Join(codeBlock, "\n")
}

func getSilentText(input string, extraOptions structs.ExtraOptions) {
	makeRequestAndGetData(input, structs.Params{ApiKey: *apiKey, ApiModel: *apiModel, Provider: *provider, Max_length: *max_length, Temperature: *temperature, Top_p: *top_p, Preprompt: *preprompt, Url: *url}, extraOptions)
}

func handleEachPart(resp *http.Response, input string) string {
	scanner := bufio.NewScanner(resp.Body)

	// Variables
	count := 0
	isCode := false
	isGreen := false
	tickCount := 0
	previousWasTick := false
	isTick := false
	isRealCode := false

	lineLength := 0
	size, termwidthErr := ts.GetSize()
	termWidth := size.Col()

	fullText := ""

	for scanner.Scan() {
		mainText := providers.GetMainText(scanner.Text(), *provider, input)
		if len(mainText) < 1 {
			continue
		}
		fullText += mainText

		if count <= 0 {
			wordLength := len([]rune(mainText))
			if termwidthErr == nil && (termWidth-lineLength < wordLength) {
				fmt.Print("\n")
				lineLength = 0
			}
			lineLength += wordLength
			splitLine := strings.Split(mainText, "")
			// Iterating through each word
			for _, word := range splitLine {
				// If its a backtick
				if word == "`" {
					tickCount++
					isTick = true

					if tickCount == 2 && !previousWasTick {
						tickCount = 0
					} else if tickCount == 6 {
						tickCount = 0
					}
					previousWasTick = true
					isGreen = false
					isCode = false

				} else {
					isTick = false
					// If its a normal word

					if tickCount == 1 {
						isGreen = true
					} else if tickCount == 3 {
						isCode = true
					}
					previousWasTick = false
				}

				if isCode {
					codeText.Print(word)
				} else if isGreen {
					boldBlue.Print(word)
				} else if !isTick {
					fmt.Print(word)
				}
			}
		} else {
			wordLength := len([]rune(mainText))

			if termwidthErr == nil && (termWidth-lineLength < wordLength) {
				fmt.Print("\n")
				lineLength = 0
			}
			lineLength += wordLength
			splitLine := strings.Split(mainText, "")

			if mainText == "``" || mainText == "```" {
				isRealCode = true
			} else {
				isRealCode = false
			}

			for _, word := range splitLine {
				// If its a backtick
				if word == "`" {
					tickCount++
					isTick = true

					if tickCount == 2 && !previousWasTick {
						tickCount = 0
					} else if tickCount >= 6 && tickCount%2 == 0 && previousWasTick {
						tickCount = 0
					}
					isGreen = false
					isCode = false

				} else {
					if word == "\n" {
						lineLength = 0
					}
					isTick = false
					// If its a normal word
					if tickCount == 1 {
						isGreen = true
					} else if tickCount >= 3 {
						isCode = true
					}
				}

				if isCode {
					codeText.Print(word)
				} else if isGreen {
					boldBlue.Print(word)
				} else if !isTick {
					fmt.Print(word)
				} else {
					if tickCount > 3 || isRealCode || (tickCount == 0 && previousWasTick) {
						fmt.Print(word)
					}

				}
				if word == "`" {
					previousWasTick = true
				} else {
					previousWasTick = false
				}

			}
		}

		count++
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred. Error:", err)
		os.Exit(1)
	}

	return fullText

}

func printConnectionErrorMsg(err error) {
	bold.Fprintln(os.Stderr, "\rSome error has occurred. Check your internet connection.")
	fmt.Fprintln(os.Stderr, "\nError:", err)
	os.Exit(1)
}

func handleStatus400(resp *http.Response) {
	bold.Fprintln(os.Stderr, "\rSome error has occurred. Statuscode:", resp.StatusCode)
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println(string(respBody))
	os.Exit(1)
}

// func generateImageCraiyon(prompt string) {
// 	bold.Println("Generating images...")
// 	client, err := client.NewClient()
// 	if err != nil {
// 		fmt.Fprintln(os.Stderr, err)
// 		os.Exit(1)
// 	}

// 	url := "https://api.craiyon.com/v3"

// 	safeInput, _ := json.Marshal(prompt)

// 	payload := strings.NewReader(fmt.Sprintf(`{
// 		"prompt": %v,
// 		"token": null,
// 		"model": "photo",
// 		"negative_prompt": "",
// 		"version": "c4ue22fb7kb6wlac"
// 	}`, string(safeInput)))

// 	req, _ := http.NewRequest("POST", url, payload)

// 	req.Header.Set("Content-Type", "application/json")
// 	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/127.0")

// 	res, err := client.Do(req)

// 	if err != nil {
// 		fmt.Fprint(os.Stderr, "Check your internet connection\n\n")
// 		fmt.Fprintln(os.Stderr, "Error:", err)
// 		os.Exit(0)
// 	}

// 	defer res.Body.Close()

// 	var responseObj ImgResponse

// 	err = json.NewDecoder(res.Body).Decode(&responseObj)
// 	if err != nil {
// 		// Handle error
// 		fmt.Fprintln(os.Stderr, "Error:", err)
// 		return
// 	}

// 	imgList := responseObj.Images

// 	fmt.Println("Saving images in current directory in folder:", prompt)
// 	if _, err := os.Stat(prompt); os.IsNotExist(err) {
// 		err := os.Mkdir(prompt, 0755)
// 		if err != nil {
// 			fmt.Fprintln(os.Stderr, err)
// 			os.Exit(1)
// 		}
// 	}

// 	for i := 0; i < len(imgList); i++ {
// 		downloadUrl := "https://img.craiyon.com/" + imgList[i]
// 		downloadImage(downloadUrl, prompt)

// 	}
// }

func downloadImage(url string, destDir string) error {
	client, err := client.NewClient()
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		// Handle error
		return err
	}

	response, err := client.Do(req)

	if err != nil {
		return err
	}
	defer response.Body.Close()

	fileName := filepath.Join(destDir, filepath.Base(url))
	file, err := os.Create(fileName)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, response.Body)
	if err != nil {
		return err
	}
	fmt.Println("Saved image", fileName)

	return nil
}

func executeCommand(shellName string, shellOptions []string, fullLine string) {
	// Directly use the shellName variable set by setShellAndOSVars()
	cmd := exec.Command(shellName, append(shellOptions, fullLine)...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	addToShellHistory(fullLine)
}

func addToShellHistory(command string) {
	shell := os.Getenv("SHELL")
	homeDir := os.Getenv("HOME")

	if strings.Contains(shell, "/bash") {
		historyPath := os.Getenv("HISTFILE")

		if historyPath == "" {
			historyPath = homeDir + "/.bash_history"
		}

		file, _ := os.OpenFile(historyPath, os.O_APPEND|os.O_WRONLY, 0644)
		// if err != nil {
		// }
		defer file.Close()

		_, _ = file.WriteString(command + "\n")
	}
}

func makeRequestAndGetData(input string, params structs.Params, extraOptions structs.ExtraOptions) string {
	resp, err := providers.NewRequest(input, params, extraOptions)

	if err != nil {
		stopSpin = true
		printConnectionErrorMsg(err)
	}

	defer resp.Body.Close()

	code := resp.StatusCode

	if code >= 400 {
		stopSpin = true
		fmt.Print("\r")
		if !extraOptions.IsInteractive {
			handleStatus400(resp)
		}
		respBody, _ := io.ReadAll(resp.Body)
		fmt.Println("Some error has occurred, try again")
		fmt.Println(string(respBody))
		return ""
	}

	stopSpin = true
	fmt.Print("\r")

	if extraOptions.IsNormal {
		// Print the Question
		if !extraOptions.IsInteractive {
			fmt.Print("\r          \r")
			// bold.Printf("\r%v\n\n", input)
			bold.Println()
		} else {
			fmt.Println()
			boldViolet.Println("╭─ Bot")
		}

		// Handling each part
		return handleEachPart(resp, input)
	}

	if extraOptions.IsGetCommand {
		fmt.Print("\r          \r")
	}

	scanner := bufio.NewScanner(resp.Body)

	// Handling each part
	fullText := ""

	for scanner.Scan() {
		mainText := providers.GetMainText(scanner.Text(), *provider, input)
		if len(mainText) < 1 {
			continue
		}
		fullText += mainText

		if !extraOptions.IsGetWhole {
			fmt.Print(mainText)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Some error has occurred. Error:", err)
		os.Exit(1)
	}

	if extraOptions.IsGetWhole {
		fmt.Println(fullText)
	}

	if extraOptions.IsGetSilent || extraOptions.IsGetCode {
		fmt.Println()
	}

	if extraOptions.IsGetCommand {
		lineCount := strings.Count(fullText, "\n") + 1

		if lineCount == 1 {
			if *shouldExecuteCommand {
				fmt.Println()
				executeCommand(shellName, shellOptions, fullText)
			} else {
				bold.Print("\n\nExecute shell command? [y/n]: ")
				reader := bufio.NewReader(os.Stdin)
				userInput, _ := reader.ReadString('\n')
				userInput = strings.TrimSpace(userInput)

				if userInput == "y" || userInput == "" {
					executeCommand(shellName, shellOptions, fullText)
				}
			}
		}
	}

	return ""
}