package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	http "github.com/bogdanfinn/fhttp"

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
		tls_client.WithCookieJar(jar), // create cookieJar instance and pass it as argument
	}
	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		fmt.Println(err)
		return
	}
	var data = strings.NewReader(fmt.Sprintf(`{"prompt":"%v","options":{"parentMessageId":"%v"}}`, input, chatId))
	req, err := http.NewRequest("POST", "https://chatbot.theb.ai/api/chat-process", data)
	if err != nil {
		fmt.Println("\nSome error has occured.")
		fmt.Println("Error:", err)
		os.Exit(0)
	}
	// Setting all the required headers
	// req.Header.Set("Host", "chatbot.theb.ai")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/110.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// req.Header.Set("Accept-Encoding", "identity")
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

	// Print the Question
	if !isInteractive {
		fmt.Print("\r         ")
		bold.Printf("\r%v\n\n", input)
	} else {
		fmt.Println()
	}

	gotId := false
	id := ""
	// Handling each json
	for scanner.Scan() {
		var jsonObj map[string]interface{}
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &jsonObj)
		if err != nil {
			bold.Println("\rError. Your IP is being blocked by the server.")
			fmt.Println("Status Code:", code)
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
			splitLine := strings.Split(result, "")

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
					previousWasTick = true
					isGreen = false
					isCode = false

				} else {
					isTick = false
					// If its a normal word
					if tickCount == 1 {
						isGreen = true
					} else if tickCount >= 3 {
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
				} else {
					if tickCount > 3 {
						fmt.Print(word)
					}

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
	fmt.Println("")
	createConfig(configDir, id)
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
	spinChars := []string{"|", "/", "-", "\\"}
	i := 0
	for {
		if *stop {
			break
		}
		fmt.Printf("\r%s Loading", spinChars[i])
		i = (i + 1) % len(spinChars)
		time.Sleep(100 * time.Millisecond)
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
			tls_client.WithCookieJar(jar), // create cookieJar instance and pass it as argument
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
			cmd := exec.Command("bash", "-c", "curl -sSL https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash")
			_, err := cmd.CombinedOutput()
			if err != nil {
				fmt.Println("Error updating.", err)
			}
			fmt.Println("Successfully updated.")

		} else {
			fmt.Println("You are already using the latest version.")
		}
	}
}
