package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

const letters string = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const VERSION = "1.2.1"

var bold = color.New(color.Bold)
var boldGreen = color.New(color.Bold, color.FgGreen)
var stopSpin = false

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

	if len(args) > 1 && len(args[1]) > 1 {
		input := args[1]

		if input == "-v" || input == "--version" {
			fmt.Println("tgpt", VERSION)
		} else if input == "-f" || input == "--forget" {
			error := os.Remove(configDir + "/tgpt/config.txt")
			if error != nil {
				fmt.Println("There is not history to remove")
			} else {
				fmt.Println("Chat history removed")
			}
		} else if strings.HasPrefix(input, "-") {
			color.Blue(`Usage: tgpt "Explain quantum computing in simple terms"`)
			boldGreen.Println("Options:")
			fmt.Printf("%-50v Forget chat history \n", "-f, --forget")
			fmt.Printf("%-50v Print version \n", "-v, --version")
			fmt.Printf("%-50v Print help message \n", "-h, --help")

			boldGreen.Println("\nExample:")
			fmt.Println("tgpt -f")
		} else {
			go loading(&stopSpin)
			formattedInput := strings.ReplaceAll(input, `"`, `\"`)
			inputLength := len(formattedInput) + 87
			getData(formattedInput, inputLength, chatId, configDir+"/tgpt")
		}

	} else {
		color.Red("You have to write some text")
		color.Blue(`Example: tgpt "Explain quantum computing in simple terms"`)
	}

}
