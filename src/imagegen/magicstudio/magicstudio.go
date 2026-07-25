package magicstudio

import (
	"bytes"
	"fmt"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	http "github.com/bogdanfinn/fhttp"
	"github.com/aandrew-me/tgpt/v2/src/client"
	"io"
	"log"
	"mime/multipart"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

func GenerateImageMagicStudio(prompt string, params structs.ImageParams) string {
	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	formField, err := writer.CreateFormField("prompt")
	if err != nil {
		log.Fatal(err)
	}
	_, err = formField.Write([]byte(prompt))

	formField, err = writer.CreateFormField("output_format")
	if err != nil {
		log.Fatal(err)
	}
	_, err = formField.Write([]byte("bytes"))

	formField, err = writer.CreateFormField("anonymous_user_id")
	if err != nil {
		log.Fatal(err)
	}

	// UUID
	randomID := uuid.New().String()
	_, err = formField.Write([]byte(randomID))

	formField, err = writer.CreateFormField("request_timestamp")
	if err != nil {
		log.Fatal(err)
	}
	ts := strconv.FormatFloat(float64(time.Now().UnixNano())/1e9, 'f', 3, 64)
	_, err = formField.Write([]byte(ts))

	formField, err = writer.CreateFormField("user_is_subscribed")
	if err != nil {
		log.Fatal(err)
	}
	_, err = formField.Write([]byte("true"))

	writer.Close()

		req, err := http.NewRequest("POST", "https://ai-api.magicstudio.com/api/ai-art-generator", form)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("accept", "application/json, text/plain, */*")
	req.Header.Set("accept-language", "en-US,en;q=0.5")
	req.Header.Set("cache-control", "no-cache")
	req.Header.Set("origin", "https://magicstudio.com")
	req.Header.Set("referer", "https://magicstudio.com/")
	req.Header.Add("User-Agent", "Mozilla/5.0 (X11; Linux x86_64; rv:127.0) Gecko/20100101 Firefox/148.0")
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client, err := client.NewClient()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	ct := resp.Header.Get("Content-Type")

	if strings.Contains(ct, "image/jpeg") || strings.Contains(ct, "image/jpg") {
		filename := fmt.Sprintf("magic_%s.jpg", randomID)
		if err := os.WriteFile(filename, bodyBytes, 0644); err != nil {
			log.Fatal(err)
		}
		return filename
	}

	return ""
}
