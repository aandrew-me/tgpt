package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log"
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

func getData(input string, inputLength int, chatId string, configDir string) {
	randomString := getRandomString(15)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(fmt.Sprintf(`{"prompt":"%v","options":{"parentMessageId":"%v"}}`, input, chatId))
	req, err := http.NewRequest("POST", "https://chatbot.theb.ai/api/chat-process", data)
	if err != nil {
		log.Fatal("Some error has occured. Code 1")
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
		log.Fatal("Some error has occured. Check your internet connection.")
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
	bold.Print(input, "\n\n")

	gotId := false
	id := ""
	// Handling each json
	for scanner.Scan() {
		var jsonObj map[string]interface{}
		line := scanner.Text()
		err := json.Unmarshal([]byte(line), &jsonObj)
		if err != nil {
			log.Fatal("Some error has occured")
		}
		mainText := fmt.Sprintf("%s", jsonObj["text"])
		if !gotId {
			id = fmt.Sprintf("%s", jsonObj["id"])
			gotId = true
		}

		if count <= 0 {
			oldLine = mainText
			splitLine := strings.Split(oldLine, "")
			for _, word := range splitLine {
				fmt.Print(word)
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
	fmt.Println("")
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	createConfig(configDir, id)
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

func createConfig(dir string, chatId string){
	err := os.MkdirAll(dir, 0755)
	configTxt := "id:" + chatId
	if err != nil {
		fmt.Println(err)
	} else {
		os.WriteFile(dir + "/config.txt", []byte(configTxt), 0755)
	}
}

func main() {
	hasConfig := true
	configDir, error := os.UserConfigDir()

	if error != nil {
		hasConfig = false
	}
	configTxtByte, err := os.ReadFile(configDir + "/tgpt/config.txt")
	if err != nil {
		hasConfig = false
	}
	chatId := ""
	if hasConfig {
		chatId = strings.Split(string(configTxtByte), ":")[1]
	}
	args := os.Args

	if len(args) > 1 {
		input := args[1]

		if input == "-h" || input == "--help" {
			color.Blue(`Usage: tgpt "Explain quantum computing in simple terms"`)
		} else {
			go loading(&stopSpin)
			formattedInput := strings.ReplaceAll(input, `"`, `\"`)
			inputLength := len(formattedInput) + 87
			getData(formattedInput, inputLength, chatId, configDir + "/tgpt")
		}

	} else {
		color.Red("You have to write some text")
		color.Blue(`Example: tgpt "Explain quantum computing in simple terms"`)
	}

}
