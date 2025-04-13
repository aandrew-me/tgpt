package utils

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"path/filepath"

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