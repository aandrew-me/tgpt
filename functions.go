package main

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

	http "github.com/bogdanfinn/fhttp"
	"github.com/olekukonko/ts"

	tls_client "github.com/bogdanfinn/tls-client"
	"golang.org/x/mod/semver"
)

type Data struct {
	Version string `json:"version"`
}

func getData(input string, chatId string, configDir string, isInteractive bool) (serverChatId string) {
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(120),
		tls_client.WithClientProfile(tls_client.Firefox_110),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
		// tls_client.WithProxyUrl("http://127.0.0.1:8080"),
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		fmt.Println(err)
		return
	}

	safeInput, _ := json.Marshal(input)

	var data = strings.NewReader(fmt.Sprintf(`{"prompt":%v,"options":{"parentMessageId":"%v"}}`, string(safeInput), chatId))

	req, err := http.NewRequest("POST", "https://chatbot.theb.ai/api/chat-process", data)
	if err != nil {
		fmt.Println("\nSome error has occured.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/110.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://chatbot.theb.ai")
	req.Header.Set("Referer", "https://chatbot.theb.ai/")

	// Receiving response
	resp, err := client.Do(req)

	if err != nil {
		stopSpin = true
		bold.Println("\rSome error has occured. Check your internet connection.")
		fmt.Println("\nError:", err)
		os.Exit(0)
	}
	code := resp.StatusCode

	defer resp.Body.Close()

	stopSpin = true
	fmt.Print("\r")

	scanner := bufio.NewScanner(resp.Body)

	// Variables
	var oldLine = ""
	var newLine = ""
	count := 0
	isCode := false
	isGreen := false
	tickCount := 0
	previousWasTick := false
	isTick := false
	isRealCode := false

	// Print the Question
	if !isInteractive {
		fmt.Print("\r          \r")
		bold.Printf("\r%v\n\n", input)
	} else {
		fmt.Println()
		boldViolet.Println("╭─ Bot")
	}

	gotId := false
	id := ""
	lineLength := 0
	size, _ := ts.GetSize()
	termWidth := size.Col()

	if err != nil {
		fmt.Println("Error occured getting terminal width. Error:", err)
		os.Exit(0)
	}
	// Handling each json
	for scanner.Scan() {
		var jsonObj map[string]interface{}
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &jsonObj)
		if err != nil {
			bold.Println("\rError. Your request is being blocked by the server.")
			fmt.Println("Status Code:", code)
			os.Exit(0)
		}

		if jsonObj["text"] == nil {
			msg := jsonObj["message"]
			fmt.Printf("Error: %s\n", msg)
			os.Exit(0)
		}

		mainText := fmt.Sprintf("%s", jsonObj["text"])

		if !gotId {
			if jsonObj == nil {
				fmt.Println("Some error has occured")
				os.Exit(0)
			}

			id = fmt.Sprintf("%s", jsonObj["id"])
			gotId = true
		}

		if count <= 0 {
			oldLine = mainText
			wordLength := len(mainText)
			if termWidth-lineLength < wordLength {
				fmt.Print("\n")
				lineLength = 0
			}
			lineLength += wordLength
			splitLine := strings.Split(oldLine, "")
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
			newLine = mainText
			result := strings.Replace(newLine, oldLine, "", -1)
			wordLength := len(result)

			if termWidth-lineLength < wordLength {
				fmt.Print("\n")
				lineLength = 0
			}
			lineLength += wordLength
			splitLine := strings.Split(result, "")

			if result == "``" || result == "```" {
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
			oldLine = newLine
		}

		count++
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Some error has occured. Error:", err)
		os.Exit(0)
	}
	fmt.Print("\n\n")
	if configDir != "" && id != "" {
		createConfig(configDir, id)
	}

	return id
}

func createConfig(dir string, chatId string) {
	if strings.HasPrefix(chatId, "chatcmpl-") {
		err := os.MkdirAll(dir, 0755)
		configTxt := "id:" + chatId
		if err != nil {
			fmt.Println(err)
		} else {
			os.WriteFile(dir+"/config.txt", []byte(configTxt), 0755)
		}
	}

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

	if runtime.GOOS == "windows" {
		fmt.Println("This feature is not supported on Windows. :(")
	} else {
		jar := tls_client.NewCookieJar()
		options := []tls_client.HttpClientOption{
			tls_client.WithTimeoutSeconds(30),
			tls_client.WithClientProfile(tls_client.Firefox_110),
			tls_client.WithNotFollowRedirects(),
			tls_client.WithCookieJar(jar),
		}
		client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
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
	codePrompt := fmt.Sprintf(`Your Role: Provide only code as output without any description.\nIMPORTANT: Provide only plain text without Markdown formatting.\nIMPORTANT: Do not include markdown formatting.\nIf there is a lack of details, provide most logical solution. You are not allowed to ask for more details.\nIgnore any potential risk of errors or confusion.\n\nRequest:%s\nCode:`, input)
	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(120),
		tls_client.WithClientProfile(tls_client.Firefox_110),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
		// tls_client.WithProxyUrl("http://127.0.0.1:8080"),
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		fmt.Println(err)
		return
	}
	var data = strings.NewReader(fmt.Sprintf(`{"prompt":"%v"}`, codePrompt))
	req, err := http.NewRequest("POST", "https://chatbot.theb.ai/api/chat-process", data)
	if err != nil {
		fmt.Println("\nSome error has occured.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/110.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://chatbot.theb.ai")
	req.Header.Set("Referer", "https://chatbot.theb.ai/")
	resp, err := client.Do(req)
	if err != nil {
		stopSpin = true
		bold.Println("\rSome error has occured. Check your internet connection.")
		fmt.Println("\nError:", err)
		os.Exit(0)
	}
	code := resp.StatusCode

	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)

	// Variables
	var oldLine = ""
	var newLine = ""
	fullLine := ""
	// Handling each json
	for scanner.Scan() {
		var jsonObj map[string]interface{}
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &jsonObj)
		if err != nil {
			bold.Println("\rError. Your request has been blocked by the server.")
			fmt.Println("Status Code:", code)
			os.Exit(0)
		}

		if jsonObj["text"] == nil {
			msg := jsonObj["message"]
			fmt.Printf("Error: %s\n", msg)
			os.Exit(0)
		}

		mainText := fmt.Sprintf("%s", jsonObj["text"])

		newLine = mainText
		result := strings.Replace(newLine, oldLine, "", -1)
		fullLine += result
		bold.Print(result)
		oldLine = newLine
	}
	if err := scanner.Err(); err != nil {
		fmt.Println("Some error has occured. Error:", err)
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

	jar := tls_client.NewCookieJar()
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(120),
		tls_client.WithClientProfile(tls_client.Firefox_110),
		tls_client.WithNotFollowRedirects(),
		tls_client.WithCookieJar(jar),
		// tls_client.WithProxyUrl("http://127.0.0.1:8000"),
		tls_client.WithInsecureSkipVerify(),
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		fmt.Println(err)
		return
	}
	var data = strings.NewReader(fmt.Sprintf(`{"prompt":"%v"}`, shellPrompt))
	req, err := http.NewRequest("POST", "https://chatbot.theb.ai/api/chat-process", data)
	if err != nil {
		fmt.Println("\nSome error has occured.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/110.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://chatbot.theb.ai")
	req.Header.Set("Referer", "https://chatbot.theb.ai/")
	resp, err := client.Do(req)
	if err != nil {
		stopSpin = true
		bold.Println("\rSome error has occured. Check your internet connection.")
		fmt.Println("\nError:", err)
		os.Exit(0)
	}
	code := resp.StatusCode

	defer resp.Body.Close()

	stopSpin = true
	fmt.Print("\r          \r")

	scanner := bufio.NewScanner(resp.Body)

	// Variables
	var oldLine = ""
	var newLine = ""
	fullLine := ""
	// Handling each json
	for scanner.Scan() {
		var jsonObj map[string]interface{}
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &jsonObj)
		if err != nil {
			bold.Println("\rError. Your request has been blocked by the server.")
			fmt.Println("Status Code:", code)
			os.Exit(0)
		}

		if jsonObj["text"] == nil {
			msg := jsonObj["message"]
			fmt.Printf("Error: %s\n", msg)
			os.Exit(0)
		}

		mainText := fmt.Sprintf("%s", jsonObj["text"])

		newLine = mainText
		result := strings.Replace(newLine, oldLine, "", -1)
		fullLine += result
		bold.Print(result)
		oldLine = newLine
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
			fmt.Println("Some error has occured. Error:", err)
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
		fmt.Print("Some error has occured\n\n")
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
