package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
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

func getData(input string, inputLength int) {
	randomString := getRandomString(15)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	var data = strings.NewReader(fmt.Sprintf(`{"prompt":"%v","options":{"parentMessageId":"chatcmpl-75z6jNw2bxwG9ATGUPQCDsZYOQX5N"}}`, input))
	req, err := http.NewRequest("POST", "https://chatbot.theb.ai/api/chat-process", data)
	if err != nil {
		log.Fatal(err)
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
		log.Fatal(err)
	}

	defer resp.Body.Close()
	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}
	stopSpin = true
	fmt.Print("\r")

	// Adding comma after each json object
	mainText := regexp.MustCompile(`}\n`).ReplaceAllString(string(bodyText), "},\n")

	// Adding brackets to make it an array
	jsonArray := fmt.Sprintf("[%s]", mainText)
	bold := color.New(color.Bold)

	bold.Print(input, "\n\n")

	var chatData []ChatData

	error := json.Unmarshal([]byte(jsonArray), &chatData)
	if error != nil {
		fmt.Println("error parsing JSON: ", err)
		return
	}

	// Selecting the last one
	text := chatData[len(chatData)-1].Text

	threeTickPattern := regexp.MustCompile("```([\\s\\S]*?)```")

	oneTickPattern := regexp.MustCompile("`([\\s\\S]*?)`")

	matches := threeTickPattern.FindAllStringSubmatch(text, -1)

	for _, match := range matches {
		capturedText := match[1]
		blueText := color.BlueString(capturedText)
		text = strings.ReplaceAll(text, match[0], blueText)
	}

	moreMatches := oneTickPattern.FindAllStringSubmatch(text, -1)

	for _, match := range moreMatches {
		capturedText := match[1]
		blueText := color.GreenString(capturedText)
		text = strings.ReplaceAll(text, match[0], blueText)
	}

	fmt.Println(text)
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


type ChatData struct {
	Text string `json:"text"`
}

func main() {
	// getData()
	args := os.Args

	if len(args) > 1 {
		input := args[1]

		if input == "-h" || input == "--help" {
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
