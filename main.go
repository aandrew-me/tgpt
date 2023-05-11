package main

import (
	"bufio"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"strings"
	"syscall"

	"github.com/fatih/color"
)

const localVersion = "1.4.3"

var bold = color.New(color.Bold)
var boldBlue = color.New(color.Bold, color.FgBlue)
var codeText = color.New(color.BgBlack, color.FgGreen)
var stopSpin = false

func main() {
	terminate := make(chan os.Signal, 1)
	signal.Notify(terminate, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-terminate
		os.Exit(0)
	}()

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
			fmt.Println("tgpt", localVersion)
		} else if input == "-u" || input == "--update" {
			update()
		} else if input == "-i" || input == "--interactive" {
			reader := bufio.NewReader(os.Stdin)
			bold.Println("Interactive mode started. Press Ctrl + C or type exit to quit.\n")
			serverID := chatId
			for {
				bold.Print(">> ")

				input, err := reader.ReadString('\n')
				if err != nil {
					fmt.Println("Error reading input:", err)
					break
				}

				if len(input) > 1 {
					input = strings.TrimSpace(input)
					if input == "exit" {
						bold.Println("Exiting...")
						return
					}
					serverID = getData(input, serverID, configDir+"/tgpt", true)

				}

			}

		} else if input == "-f" || input == "--forget" {
			error := os.Remove(configDir + "/tgpt/config.txt")
			if error != nil {
				fmt.Println("There is no history to remove")
			} else {
				fmt.Println("Chat history removed")
			}
		} else if strings.HasPrefix(input, "-") {
			color.Blue(`Usage: tgpt "Explain quantum computing in simple terms"`)
			boldBlue.Println("Options:")
			fmt.Printf("%-50v Forget chat history \n", "-f, --forget")
			fmt.Printf("%-50v Print version \n", "-v, --version")
			fmt.Printf("%-50v Print help message \n", "-h, --help")
			fmt.Printf("%-50v Start interactive mode \n", "-i, --interactive")
			if runtime.GOOS != "windows" {
				fmt.Printf("%-50v Update program \n", "-u, --update")
			}

			boldBlue.Println("\nExample:")
			fmt.Println("tgpt -f")
		} else {
			go loading(&stopSpin)
			formattedInput := strings.ReplaceAll(input, `"`, `\"`)
			getData(formattedInput, chatId, configDir+"/tgpt", false)
		}

	} else {
		color.Red("You have to write some text")
		color.Blue(`Example: tgpt "Explain quantum computing in simple terms"`)
	}
}
