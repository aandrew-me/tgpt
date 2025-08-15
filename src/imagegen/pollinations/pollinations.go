package pollinations_img

import (
	"fmt"
	"io"
	"log"
	"os"
	"strconv"

	url_package "net/url"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
)

func GenerateImagePollinations(prompt string, params structs.ImageParams) string {
	client, err := client.NewClient()

	if err != nil {
		log.Fatal(err)
	}

	full_prompt := url_package.QueryEscape(prompt)

	filepath := params.Out
	if filepath == "" {
		randId := utils.RandomString(20)
		filepath = randId + ".jpg"
	}

	model := "flux"
	if params.ApiModel != "" {
		model = params.ApiModel
	}

	link := fmt.Sprintf("https://image.pollinations.ai/prompt/%v", full_prompt)

	queryParams := url_package.Values{}

	seed := utils.GenerateRandomNumber(5)

	width := strconv.Itoa(params.Width)
	if width == "" {
		width = "1024"
	}

	height := strconv.Itoa(params.Height)
	if height == "" {
		height = "1024"
	}

	queryParams.Add("model", model)
	queryParams.Add("width", width)
	queryParams.Add("height", height)
	queryParams.Add("nologo", "true")
	queryParams.Add("seed", seed)
	queryParams.Add("private", "true")
	queryParams.Add("enhance", "true")
	queryParams.Add("referer", "tgpt")

	urlObj, err := url_package.Parse(link)
	if err != nil {
		log.Fatal("Error parsing URL:", err)
	}

	urlObj.RawQuery = queryParams.Encode()

	req, _ := http.NewRequest("GET", urlObj.String(), nil)

	req.Header.Add("Referer", "tgpt")

	res, err := client.Do(req)

	if err != nil {
		log.Fatal(os.Stderr, err)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		responseText := string(body)

		log.Fatalf("Some error has occurred. Try again (perhaps with a different model).\nError: %v", responseText)
	}

	file, err := os.Create(filepath)
	if err != nil {
		log.Fatal(os.Stderr, "Error: %v", err)
	}
	defer file.Close()

	// Copy the response body (image data) to the file
	_, err = io.Copy(file, res.Body)

	if err != nil {
		log.Fatal(os.Stderr, "Error: %v", err)
	}

	return filepath
}
