package utils

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/aandrew-me/tgpt/v2/src/client"
	http "github.com/bogdanfinn/fhttp"

	"github.com/fatih/color"
)

func RandomString(length int) string {
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = characters[rand.Intn(len(characters))]
	}
	return string(result)
}

func GenerateRandomNumber(length int) string {
	numbers := []rune("0123456789")
	result := make([]rune, length)
	for i := range result {
		result[i] = numbers[rand.Intn(len(numbers))]
	}

	return string(result)
}

func LogToFile(text string, logType string, logPath string) {
	logTxt := logType + ": " + text

	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)

	}
	defer file.Close()

	_, err = file.WriteString(logTxt + "\n\n")
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)

	}
}

func PrintError(text string) {
	red := color.New(color.FgRed)

	red.Fprintln(os.Stderr, text)
}

func DownloadImage(url string, destDir string) error {
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

func OpenImage(path string) error {
	switch runtime.GOOS {
	case "linux":
		return exec.Command("xdg-open", path).Start()
	case "darwin":
		return exec.Command("open", path).Start()
	case "windows":
		return exec.Command("cmd", "/c", "start", "", path).Start()
	default:
		return exec.ErrNotFound
	}
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
			}
			capturing = true
			continue
		}
		if capturing {
			codeBlock = append([]string{lines[i]}, codeBlock...)
		}
	}

	if capturing || len(codeBlock) == 0 {
		return ""
	}

	return strings.Join(codeBlock, "\n")
}