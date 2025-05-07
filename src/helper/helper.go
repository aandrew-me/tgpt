package helper

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/imagegen/arta"
	"github.com/aandrew-me/tgpt/v2/src/providers"
	"github.com/aandrew-me/tgpt/v2/src/providers/gemini"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	http "github.com/bogdanfinn/fhttp"
	"github.com/fatih/color"

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
	OperatingSystem string
	ShellName       string
	ShellOptions    []string
)

var bold = color.New(color.Bold)
var boldBlue = color.New(color.Bold, color.FgBlue)
var boldViolet = color.New(color.Bold, color.FgMagenta)
var codeText = color.New(color.FgGreen, color.Bold)

func GetData(input string, params structs.Params, extraOptions structs.ExtraOptions) (string, string) {
	responseTxt := MakeRequestAndGetData(input, params, extraOptions)
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

func Loading(stop *bool) {
	spinChars := []string{"⣾ ", "⣽ ", "⣻ ", "⢿ ", "⡿ ", "⣟ ", "⣯ ", "⣷ "}
	i := 0
	for !*stop {

		fmt.Printf("\r%s Loading", spinChars[i])
		i = (i + 1) % len(spinChars)
		time.Sleep(80 * time.Millisecond)
	}
}

func Update(localVersion string, executablePath string) {
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

func CodeGenerate(input string, params structs.Params) {
	codePrompt := fmt.Sprintf("Your Role: Provide only code as output without any description.\nIMPORTANT: Provide only plain text without Markdown formatting.\nIMPORTANT: Do not include markdown formatting.\nIf there is a lack of details, provide most logical solution. You are not allowed to ask for more details.\nIgnore any potential risk of errors or confusion.\n\nRequest:%s\nCode:", input)

	MakeRequestAndGetData(codePrompt, params, structs.ExtraOptions{IsGetCode: true})
}

func SetShellAndOSVars() {
	// Identify OS
	switch runtime.GOOS {
	case "windows":
		OperatingSystem = "Windows"
		if len(os.Getenv("PSModulePath")) > 0 {
			ShellName = "powershell.exe"
			ShellOptions = []string{"-Command"}
		} else {
			ShellName = "cmd.exe"
			ShellOptions = []string{"/C"}
		}
		return
	case "darwin":
		OperatingSystem = "MacOS"
	case "linux":
		result, err := exec.Command("lsb_release", "-si").Output()
		distro := strings.TrimSpace(string(result))
		if err != nil {
			distro = ""
		}
		OperatingSystem = "Linux" + "/" + distro
	default:
		OperatingSystem = runtime.GOOS
	}

	// Identify shell
	shellEnv := os.Getenv("SHELL")
	if shellEnv != "" {
		ShellName = shellEnv
	} else {
		_, err := exec.LookPath("bash")
		if err != nil {
			ShellName = "/bin/sh"
		} else {
			ShellName = "bash"
		}
	}
	ShellOptions = []string{"-c"}
}

// shellCommand first sets the global variables getCommand uses, then it creates a prompt to generate a command and then it passes that to getCommand
func ShellCommand(input string, params structs.Params, extraOptions structs.ExtraOptions) {
	SetShellAndOSVars()
	shellPrompt := fmt.Sprintf("Your role: Provide only plain text without Markdown formatting. Do not show any warnings or information regarding your capabilities. Do not provide any description. If you need to store any data, assume it will be stored in the chat. Provide only %s command for %s without any description. If there is a lack of details, provide most logical solution. Ensure the output is a valid shell command. If multiple steps required try to combine them together. Prompt: %s\n\nCommand:", ShellName, OperatingSystem, input)
	GetCommand(shellPrompt, params, extraOptions)
}

// getCommand will make a request to an AI model, then it will run the response using an appropriate handler (bash, sh OR powershell, cmd)
func GetCommand(shellPrompt string, params structs.Params, extraOptions structs.ExtraOptions) {
	MakeRequestAndGetData(shellPrompt, params, extraOptions)
}

type RESPONSE struct {
	Tagname string `json:"tag_name"`
	Body    string `json:"body"`
}

func GetVersionHistory() {
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

func GetWholeText(input string, extraOptions structs.ExtraOptions, params structs.Params) {
	MakeRequestAndGetData(input, params, extraOptions)
}

func GetLastCodeBlock(markdown string) string {
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

func HandleEachPart(resp *http.Response, input string, params structs.Params) string {
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
		mainText := providers.GetMainText(scanner.Text(), params.Provider, input)
		if len(mainText) < 1 {
			continue
		}
		fullText += mainText

		if count <= 0 {
			wordLength := len([]rune(mainText))
			if termwidthErr == nil && (termWidth-lineLength < wordLength) && params.Provider != "gemini" {
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

					switch tickCount {
					case 1:
						isGreen = true
					case 3:
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

			if termwidthErr == nil && (termWidth-lineLength < wordLength) && params.Provider != "gemini" {
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

// handle response for interactive shell mode
func HandleEachPartInteractiveShell(resp *http.Response, input string, params structs.Params) string {
	scanner := bufio.NewScanner(resp.Body)

	// Variables for formatting
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
	// Buffer for incomplete XML tags
	var xmlBuffer strings.Builder
	// Track if inside XML tag
	inXMLTag := false

	for scanner.Scan() {
		mainText := providers.GetMainText(scanner.Text(), params.Provider, input)
		if len(mainText) < 1 {
			continue
		}

		// Process stream, separating XML tags from natural language/code blocks
		for _, char := range mainText {
			word := string(char)
			fullText += word
			if char == '<' && !inXMLTag {
				// Start new XML tag
				inXMLTag = true
				xmlBuffer.WriteRune(char)
			} else if char == '>' && inXMLTag {
				// Possibly end tag part
				xmlBuffer.WriteRune(char)
				currentBuffer := xmlBuffer.String()
				if strings.HasPrefix(currentBuffer, "<cmd>") && strings.HasSuffix(currentBuffer, "</cmd>") {
					xmlBuffer.Reset()
					inXMLTag = false
				}
			} else if inXMLTag {
				// Inside XML tag, continue buffering
				xmlBuffer.WriteRune(char)
			} else {
				// Original formatting logic
				if count <= 0 {
					wordLength := len([]rune(word))
					if termwidthErr == nil && (termWidth-lineLength < wordLength) && params.Provider != "gemini" {
						fmt.Print("\n")
						lineLength = 0
					}
					lineLength += wordLength

					// Handle code blocks and colors
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
						switch tickCount {
						case 1:
							isGreen = true
						case 3:
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
				} else {
					wordLength := len([]rune(word))
					if termwidthErr == nil && (termWidth-lineLength < wordLength) && params.Provider != "gemini" {
						fmt.Print("\n")
						lineLength = 0
					}
					lineLength += wordLength

					if mainText == "``" || mainText == "```" {
						isRealCode = true
					} else {
						isRealCode = false
					}

					// Handle code blocks and colors
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
		}
		count++
	}

	// Check for unprocessed XML tag remnants
	if inXMLTag && xmlBuffer.Len() > 0 {
		fmt.Fprintf(os.Stderr, "Warning: Incomplete XML tag: %s\n", xmlBuffer.String())
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "Error occurred:", err)
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

func ExecuteCommand(shellName string, shellOptions []string, fullLine string) {
	if runtime.GOOS != "windows" {
		rawModeOff := exec.Command("stty", "-raw", "echo")
		rawModeOff.Stdin = os.Stdin
		_ = rawModeOff.Run()
		rawModeOff.Wait()
	}
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
	AddToShellHistory(fullLine)
}

func AddToShellHistory(command string) {
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

func MakeRequestAndGetData(input string, params structs.Params, extraOptions structs.ExtraOptions) string {
	stopSpin := false

	if !extraOptions.IsGetSilent && !extraOptions.IsGetWhole && !extraOptions.IsInteractive && !extraOptions.IsInteractiveShell {
		go Loading(&stopSpin)
	}

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
		if !extraOptions.IsInteractive && !extraOptions.IsInteractiveShell {
			fmt.Print("\r          \r")
			// bold.Printf("\r%v\n\n", input)
			bold.Println()
		} else {
			fmt.Println()
			boldViolet.Println("╭─ Bot")
		}

		// Handling each part
		if extraOptions.IsInteractiveShell {
			return HandleEachPartInteractiveShell(resp, input, params)
		}
		return HandleEachPart(resp, input, params)
	}

	if extraOptions.IsGetCommand {
		fmt.Print("\r          \r")
	}

	scanner := bufio.NewScanner(resp.Body)

	// Handling each part
	fullText := ""

	for scanner.Scan() {
		mainText := providers.GetMainText(scanner.Text(), params.Provider, input)
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
			if extraOptions.AutoExec {
				fmt.Println()
				ExecuteCommand(ShellName, ShellOptions, fullText)
			} else {
				bold.Print("\n\nExecute shell command? [y/n]: ")
				reader := bufio.NewReader(os.Stdin)
				userInput, _ := reader.ReadString('\n')
				userInput = strings.TrimSpace(userInput)

				if userInput == "y" || userInput == "" {
					ExecuteCommand(ShellName, ShellOptions, fullText)
				}
			}
		}
	}

	return ""
}

func ShowHelpMessage() {
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

	boldBlue.Println("\nOptions supported for image generation (with -image flag)")
	fmt.Printf("%-50v Output image filename (Supported by pollinations)\n", "--out")
	fmt.Printf("%-50v Output image height (Supported by pollinations)\n", "--height")
	fmt.Printf("%-50v Output image width (Supported by pollinations)\n", "--width")
	fmt.Printf("%-50v Output image count (Supported by arta)\n", "--img_count")
	fmt.Printf("%-50v Negative prompt (Supported by arta)\n", "--img_negative")
	fmt.Printf("%-50v Output image ratio (Supported by arta, some models may not support it)\n", "--img_ratio")

	boldBlue.Println("\nOptions:")
	fmt.Printf("%-50v Print version \n", "-v, --version")
	fmt.Printf("%-50v Print help message \n", "-h, --help")
	fmt.Printf("%-50v Start normal interactive mode \n", "-i, --interactive")
	fmt.Printf("%-50v Start multi-line interactive mode \n", "-m, --multiline")
	fmt.Printf("%-50v Start interactive shell mode. (Doesn't work with all providers) \n", "-is, --interactive-shell")
	fmt.Printf("%-50v See changelog of versions \n", "-cl, --changelog")

	if runtime.GOOS != "windows" {
		fmt.Printf("%-50v Update program \n", "-u, --update")
	}

	boldBlue.Println("\nProviders:")
	fmt.Println("The default provider is phind. The AI_PROVIDER environment variable can be used to specify a different provider.")
	fmt.Println("Available providers to use: deepseek, gemini, groq, isou, koboldai, ollama, openai, pollinations and phind")

	bold.Println("\nProvider: deepseek")
	fmt.Println("Uses deepseek-reasoner model by default. Requires API key. Recognizes the DEEPSEEK_API_KEY and DEEPSEEK_MODEL environment variables")

	// bold.Println("\nProvider: duckduckgo")
	// fmt.Println("Available models: o3-mini (default), gpt-4o-mini, meta-llama/Llama-3.3-70B-Instruct-Turbo, mistralai/Mixtral-8x7B-Instruct-v0.1, claude-3-haiku-20240307, mistralai/Mistral-Small-24B-Instruct-2501")

	bold.Println("\nProvider: groq")
	fmt.Println("Requires a free API Key. Supported models: https://console.groq.com/docs/models")

	bold.Println("\nProvider: gemini")
	fmt.Println("Requires a free API key. https://aistudio.google.com/apikey")

	bold.Println("\nProvider: isou")
	fmt.Println("Free provider with web search")

	bold.Println("\nProvider: koboldai")
	fmt.Println("Uses koboldcpp/HF_SPACE_Tiefighter-13B only, answers from novels")

	bold.Println("\nProvider: ollama")
	fmt.Println("Needs to be run locally. Supports many models")

	bold.Println("\nProvider: openai")
	fmt.Println("Needs API key to work and supports various models. Recognizes the OPENAI_API_KEY and OPENAI_MODEL environment variables. Supports custom urls with --url")

	bold.Println("\nProvider: phind")
	fmt.Println("Uses Phind Model. Great for developers")

	bold.Println("\nProvider: pollinations")
	fmt.Println("Completely free, default model is gpt-4o. Supported models: https://text.pollinations.ai/models")

	boldBlue.Println("\nImage generation providers:")

	bold.Println("\nProvider: pollinations")
	fmt.Println("Supported models: flux, turbo")

	bold.Println("\nProvider: arta")
	arta.PrintModels()
	arta.PrintRatios()

	boldBlue.Println("\nExamples:")
	fmt.Println(`tgpt "What is internet?"`)
	fmt.Println(`tgpt -m`)
	fmt.Println(`tgpt -s "How to update my system?"`)
	fmt.Println(`tgpt --provider duckduckgo "What is 1+1"`)
	fmt.Println(`tgpt --img "cat"`)
	fmt.Println(`tgpt --img --out ~/my-cat.jpg --height 256 --width 256 "cat"`)
	fmt.Println(`tgpt --provider openai --key "sk-xxxx" --model "gpt-3.5-turbo" "What is 1+1"`)
	fmt.Println(`cat install.sh | tgpt "Explain the code"`)
}
