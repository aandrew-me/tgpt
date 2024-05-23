package utils

import (
	"fmt"
	"math/rand"
	"os"
)

func RandomString(length int) string {
	characters := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-"
	result := make([]byte, length)

	for i := 0; i < length; i++ {
		result[i] = characters[rand.Intn(len(characters))]
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
