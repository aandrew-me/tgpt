package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/fatih/color"
)

const letters string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var stopSpin = false

func getRandomString(n int) string {
	fullString := ""

	for i := 0; i < n; i++ {
		random := rand.Intn(51)
		fullString += (string(letters[random]))
	}
	return fullString
}

type ChatData struct {
	Text string `json:"text"`
}

func getData(input string, inputLength int) {
	randomString := getRandomString(15)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(fmt.Sprintf(`{"prompt":"%v","options":{"parentMessageId":"chatcmpl-75z6jNw2bxwG9ATGUPQCDsZYOQX5N"}}`, input))
	req, err := http.NewRequest("POST", "https://chatbot.theb.ai/api/chat-process", data)
	if err != nil {
		fmt.Println("\nSome error has occured. Code 1")
	}
	// Setting all the required headers
	req.Header.Set("Host", "chatbot.theb.ai")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:109.0) Gecko/20100101 Firefox/112.0")
	req.Header.Set("Accept", "application/json, text/plain, */*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.5")
	// req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Length", fmt.Sprintf(`"%v"`, inputLength))
	req.Header.Set("Origin", "https://chatbot.theb.ai")
	req.Header.Set("Referer", "https://chatbot.theb.ai/")
	req.Header.Set("Cookie", "__cf_bm="+randomString)
	resp, err := client.Do(req)
	if err != nil {
		stopSpin = true
		fmt.Println("\rSome error has occured. Check your internet connection.")
		os.Exit(0)
	}

	defer resp.Body.Close()

	stopSpin = true
	fmt.Print("\r")

	scanner := bufio.NewScanner(resp.Body)

	// Variables
	var oldLine = ""
	var newLine = ""
	count := 0
	bold := color.New(color.Bold)
	boldGreen := color.New(color.Bold, color.FgGreen)
	isCode := false
	isGreen := false
	tickCount := 0
	previousWasTick := false
	isTick := false

	// Print the Question
	fmt.Print("\r         ")
	bold.Printf("\r%v\n\n", input)

	// Handling each json
	for scanner.Scan() {
		var jsonObj map[string]interface{}
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &jsonObj)
		if err != nil {
			fmt.Println("\rSome error has occured")
			os.Exit(0)
		}
		mainText := fmt.Sprintf("%s", jsonObj["text"])

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
					previousWasTick = false
					if tickCount == 1 {
						isGreen = true
					} else if tickCount == 3 {
						isCode = true
					}
				}

				if isCode {
					fmt.Print(color.BlueString(word))
				} else if isGreen {
					boldGreen.Print(word)
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
					} else if tickCount == 6 {
						tickCount = 0
					}
					previousWasTick = true
					isGreen = false
					isCode = false

				} else {
					isTick = false
					// If its a normal word
					previousWasTick = false
					if tickCount == 1 {
						isGreen = true
					} else if tickCount == 3 {
						isCode = true
					}
				}

				if isCode {
					fmt.Print(color.BlueString(word))
				} else if isGreen {
					boldGreen.Print(word)
				} else if !isTick {
					fmt.Print(word)
				}

			}
			oldLine = newLine
		}

		count++
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	fmt.Println("")
}

func loading(stop *bool) {
	spinChars := []string{"|", "/", "-", "\\"}
	i := 0
	for {
		if *stop {
			break
		}
		fmt.Printf("\rLoading %s", spinChars[i])
		i = (i + 1) % len(spinChars)
		time.Sleep(80 * time.Millisecond)
	}
}

func main() {
	args := os.Args

	if len(args) > 1 && len(args[1]) > 1 {
		input := args[1]

		if strings.HasPrefix(input, "-") {
			color.Blue(`Usage: tgpt "Explain quantum computing in simple terms"`)
		} else {
			go loading(&stopSpin)
			formattedInput := strings.ReplaceAll(input, `"`, `\"`)
			inputLength := len(formattedInput) + 87
			getData(formattedInput, inputLength)
		}

	} else {
		color.Red("You have to write some text")
		color.Blue(`Example: tgpt "Explain quantum computing in simple terms"`)
	}

}
