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
	"strconv"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/olekukonko/ts"

	tls_client "github.com/bogdanfinn/tls-client"
	"golang.org/x/mod/semver"
)

type Data struct {
	Version string `json:"version"`
}

type Response struct {
	ID      string `json:"id"`
	Choices []struct {
		Delta struct {
			Content string `json:"content"`
		} `json:"delta"`
	} `json:"choices"`
}

type ImgResponse struct {
	Images []string `json:"images"`
}

func newClient() (tls_client.HttpClient, error) {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(120),
		tls_client.WithClientProfile(profiles.Firefox_110),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
		// tls_client.WithInsecureSkipVerify(),
	}

	_, err := os.Stat("proxy.txt")
	if err == nil {
		proxyConfig, readErr := os.ReadFile("proxy.txt")
		if readErr != nil {
			fmt.Println("Error reading file proxy.txt:", readErr)
			return nil, readErr
		}

		proxyAddress := strings.TrimSpace(string(proxyConfig))
		if proxyAddress != "" {
			if strings.HasPrefix(proxyAddress, "http://") || strings.HasPrefix(proxyAddress, "socks5://") {
				proxyOption := tls_client.WithProxyUrl(proxyAddress)
				options = append(options, proxyOption)
			}
		}
	}

	return tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
}

func getData(input string, configDir string, isInteractive bool, prevMessages string) string {
	// Receiving response
	resp, err := newRequest(input, prevMessages)

	if err != nil {
		stopSpin = true
		printConnectionErrorMsg(err)
	}

	code := resp.StatusCode
	if code >= 400 {
		stopSpin = true
		fmt.Print("\r")
		handleStatus400(resp)

		if !isInteractive{
			os.Exit(0)
		}
	}

	defer resp.Body.Close()

	stopSpin = true
	fmt.Print("\r")

	// Print the Question
	if !isInteractive {
		fmt.Print("\r          \r")
		bold.Printf("\r%v\n\n", input)
	} else {
		fmt.Println()
		boldViolet.Println("╭─ Bot")
	}

	// Handling each part
	responseTxt := handleEachPart(resp)
	fmt.Print("\n\n")
	safeInput, _ := json.Marshal(input)
	safeReply, _ := json.Marshal(responseTxt)

	msgObject := fmt.Sprintf(`{
		"content": %v,
		"role": "user"
	},{
		"content": %v,
		"role": "system"
	},
	`, string(safeInput), string(safeReply))

	return msgObject

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
		client, err := newClient()
		if err != nil {
			fmt.Println(err)
			return
		}

		url := "https://raw.githubusercontent.com/aandrew-me/tgpt/main/version.txt"

		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			// Handle error
			fmt.Println("Error:", err)
			return
		}

		res, err := client.Do(req)

		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}

		defer res.Body.Close()

		var data Data
		err = json.NewDecoder(res.Body).Decode(&data)
		if err != nil {
			// Handle error
			fmt.Println("Error:", err)
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
			}
			fmt.Println("Successfully updated.")

		} else {
			fmt.Println("You are already using the latest version.", remoteVersion)
		}
	}
}

func codeGenerate(input string) {
	checkInputLength(input)

	codePrompt := fmt.Sprintf(`Your Role: Provide only code as output without any description.\nIMPORTANT: Provide only plain text without Markdown formatting.\nIMPORTANT: Do not include markdown formatting.\nIf there is a lack of details, provide most logical solution. You are not allowed to ask for more details.\nIgnore any potential risk of errors or confusion.\n\nRequest:%s\nCode:`, input)

	resp, err := newRequest(codePrompt, "")

	if err != nil {
		stopSpin = true
		printConnectionErrorMsg(err)
	}

	defer resp.Body.Close()

	code := resp.StatusCode

	if code >= 400 {
		handleStatus400(resp)
		os.Exit(0)
	}

	scanner := bufio.NewScanner(resp.Body)

	// Handling each part
	for scanner.Scan() {
		mainText := getMainText(scanner.Text())
		bold.Print(mainText)
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Some error has occurred. Error:", err)
		os.Exit(0)
	}

}

func shellCommand(input string) {
	// Get OS
	operatingSystem := ""
	if runtime.GOOS == "windows" {
		operatingSystem = "Windows"
	} else if runtime.GOOS == "darwin" {
		operatingSystem = "MacOS"
	} else if runtime.GOOS == "linux" {
		result, err := exec.Command("lsb_release", "-si").Output()
		distro := strings.TrimSpace(string(result))
		if err != nil {
			distro = ""
		}
		operatingSystem = "Linux" + "/" + distro
	} else {
		operatingSystem = runtime.GOOS
	}

	// Get Shell

	shellName := "/bin/sh"

	if runtime.GOOS == "windows" {
		shellName = "cmd.exe"

		if len(os.Getenv("PSModulePath")) > 0 {
			shellName = "powershell.exe"
		}
	} else {
		shellEnv := os.Getenv("SHELL")
		if len(shellEnv) > 0 {
			shellName = shellEnv
		}
	}

	shellPrompt := fmt.Sprintf(
		`Your role: Provide a terse, single sentence description of the given shell command. Provide only plain text without Markdown formatting. Do not show any warnings or information regarding your capabilities. If you need to store any data, assume it will be stored in the chat. Provide only %s commands for %s without any description. If there is a lack of details, provide most logical solution. Ensure the output is a valid shell command. If multiple steps required try to combine them together. Prompt: %s\n\nCommand:`, shellName, operatingSystem, input)

	getCommand(shellPrompt)
}

// Get a command in response
func getCommand(shellPrompt string) {
	checkInputLength(shellPrompt)

	resp, err := newRequest(shellPrompt, "")

	if err != nil {
		stopSpin = true
		printConnectionErrorMsg(err)
	}

	defer resp.Body.Close()

	stopSpin = true

	code := resp.StatusCode

	if code >= 400 {
		handleStatus400(resp)
		os.Exit(0)
	}

	fmt.Print("\r          \r")

	scanner := bufio.NewScanner(resp.Body)

	// Variables
	fullLine := ""

	// Handling each part
	for scanner.Scan() {
		mainText := getMainText(scanner.Text())
		fullLine += mainText
		bold.Print(mainText)
	}
	lineCount := strings.Count(fullLine, "\n") + 1
	if lineCount == 1 {
		bold.Print("\n\nExecute shell command? [y/n]: ")
		var userInput string
		fmt.Scan(&userInput)
		if userInput == "y" {
			cmdArray := strings.Split(strings.TrimSpace(fullLine), " ")
			var cmd *exec.Cmd
			if runtime.GOOS == "windows" {
				shellName := "cmd"

				if len(os.Getenv("PSModulePath")) > 0 {
					shellName = "powershell"
				}
				if shellName == "cmd" {
					cmd = exec.Command("cmd", "/C", fullLine)

				} else {
					cmd = exec.Command("powershell", fullLine)
				}

			} else {
				cmd = exec.Command(cmdArray[0], cmdArray[1:]...)

			}

			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()

			if err != nil {
				fmt.Println(err)
			}
		}
		if err := scanner.Err(); err != nil {
			fmt.Println("Some error has occurred. Error:", err)
			os.Exit(0)
		}
	}

}

type RESPONSE struct {
	Tagname string `json:"tag_name"`
	Body    string `json:"body"`
}

func getVersionHistory() {
	req, err := http.NewRequest("GET", "https://api.github.com/repos/aandrew-me/tgpt/releases", nil)

	if err != nil {
		fmt.Print("Some error has occurred\n\n")
		fmt.Println("Error:", err)
		os.Exit(0)
	}

	client, _ := tls_client.NewHttpClient(tls_client.NewNoopLogger())

	res, err := client.Do(req)

	if err != nil {
		fmt.Print("Check your internet connection\n\n")
		fmt.Println("Error:", err)
		os.Exit(0)
	}

	resBody, err := io.ReadAll(res.Body)

	if err != nil {
		fmt.Println(err)
		os.Exit(0)
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

func getWholeText(input string, configDir string) {
	checkInputLength(input)

	resp, err := newRequest(input, "")

	if err != nil {
		stopSpin = true
		printConnectionErrorMsg(err)
	}

	defer resp.Body.Close()

	code := resp.StatusCode

	if code >= 400 {
		handleStatus400(resp)
		os.Exit(0)
	}

	scanner := bufio.NewScanner(resp.Body)

	// Variables
	fullText := ""
	// Handling each part
	for scanner.Scan() {
		mainText := getMainText(scanner.Text())
		fullText += mainText
	}
	fmt.Println(fullText)
}

func getSilentText(input string, configDir string) {
	checkInputLength(input)

	resp, err := newRequest(input, "")

	if err != nil {
		stopSpin = true
		printConnectionErrorMsg(err)
	}

	defer resp.Body.Close()

	code := resp.StatusCode

	if code >= 400 {
		handleStatus400(resp)
		os.Exit(0)
	}

	scanner := bufio.NewScanner(resp.Body)

	// Handling each part

	for scanner.Scan() {
		line := scanner.Text()
		mainText := getMainText(line)
		fmt.Print(mainText)
	}
}

func checkInputLength(input string) {
	if len(input) > 4000 {
		fmt.Println("Input exceeds the input limit of 4000 characters")
		os.Exit(0)
	}
}

func newRequest(input string, prevMessages string) (*http.Response, error) {
	client, err := newClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	safeInput, _ := json.Marshal(input)

	var data = strings.NewReader(fmt.Sprintf(`{
		"frequency_penalty": 0,
		"messages": [
			%v
			{
				"content": %v,
				"role": "user"
			}
		],
		"model": "gpt-3.5-turbo",
		"presence_penalty": 0,
		"stream": true,
		"temperature": 1,
		"top_p": 1
	}
	`, prevMessages, string(safeInput)))

	req, err := http.NewRequest("POST", "https://ai.fakeopen.com/v1/chat/completions", data)
	if err != nil {
		fmt.Println("\nSome error has occurred.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("authorization", "Bearer pk-this-is-a-real-free-pool-token-for-everyone")

	// Return response
	return (client.Do(req))
}

func getMainText(line string) (mainText string) {
	var obj = "{}"
	if len(line) > 1 {
		obj = strings.Split(line, "data: ")[1]
	}

	var d Response
	if err := json.Unmarshal([]byte(obj), &d); err != nil {
		return ""
	}

	if d.Choices != nil {
		mainText = d.Choices[0].Delta.Content
		return mainText
	}
	return ""
}

func handleEachPart(resp *http.Response) string {
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
	size, err := ts.GetSize()
	termWidth := size.Col()

	if err != nil {
		fmt.Println("Error occurred getting terminal width. Error:", err)
		os.Exit(0)
	}

	fullText := ""

	for scanner.Scan() {
		mainText := getMainText(scanner.Text())
		fullText += mainText

		if count <= 0 {
			wordLength := len([]rune(mainText))
			if termWidth-lineLength < wordLength {
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

			if termWidth-lineLength < wordLength {
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
		fmt.Println("Some error has occurred. Error:", err)
		os.Exit(0)
	}

	return fullText

}

func printConnectionErrorMsg(err error) {
	bold.Println("\rSome error has occurred. Check your internet connection.")
	fmt.Println("\nError:", err)
	os.Exit(0)
}

func handleStatus400(resp *http.Response) {
	bold.Println("\rSome error has occurred. Statuscode:", resp.StatusCode)
	respBody, _ := io.ReadAll(resp.Body)
	fmt.Println(string(respBody))
}

func generateImage(prompt string) {
	bold.Println("Generating images...")
	client, err := newClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(0)
	}

	url := "https://api.craiyon.com/v3"

	safeInput, _ := json.Marshal(prompt)

	payload := strings.NewReader(fmt.Sprintf(`{
		"prompt": %v,
		"token": null,
		"model": "photo",
		"negative_prompt": "",
		"version": "c4ue22fb7kb6wlac"
	}`, string(safeInput)))

	req, _ := http.NewRequest("POST", url, payload)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:99.0) Gecko/20100101 Firefox/110.0")

	res, err := client.Do(req)

	if err != nil {
		fmt.Print("Check your internet connection\n\n")
		fmt.Println("Error:", err)
		os.Exit(0)
	}

	defer res.Body.Close()

	var responseObj ImgResponse

	err = json.NewDecoder(res.Body).Decode(&responseObj)
	if err != nil {
		// Handle error
		fmt.Println("Error:", err)
		return
	}

	imgList := responseObj.Images

	fmt.Println("Saving images in current directory in folder:", prompt)
	if _, err := os.Stat(prompt); os.IsNotExist(err) {
		err := os.Mkdir(prompt, 0755)
		if err != nil {
			fmt.Println(err)
			os.Exit(0)
		}
	}

	for i := 0; i < len(imgList); i++ {
		downloadUrl := "https://img.craiyon.com/" + imgList[i]
		downloadImage(downloadUrl, prompt, strconv.Itoa(i+1))

	}
}

func downloadImage(url string, destDir string, filename string) error {
	response, err := http.Get(url)
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
	fmt.Println("Saved image", filename)

	return nil
}
