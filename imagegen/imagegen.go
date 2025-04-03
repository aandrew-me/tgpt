package imagegen

import (
	"fmt"
	"io"
	"os"
	"strconv"

	url_package "net/url"

	http "github.com/bogdanfinn/fhttp"

	"github.com/aandrew-me/tgpt/v2/client"
	"github.com/aandrew-me/tgpt/v2/structs"
	"github.com/aandrew-me/tgpt/v2/utils"
	"github.com/fatih/color"
)

var bold = color.New(color.Bold)

func GenerateImg(prompt string, params structs.ImageParams, isQuite bool) {
	if params.Provider == "pollinations" || params.Provider == "" {
		if !isQuite {
			bold.Println("Generating image with pollinations.ai...")
		}
		filename := generateImagePollinations(prompt, params)
		if !isQuite {
			fmt.Printf("Saved image as %v\n", filename)
		}

	} else {
		fmt.Fprintln(os.Stderr, "Such a provider doesn't exist")
		os.Exit(1)
	}
}

func generateImagePollinations(prompt string, params structs.ImageParams) string {

	client, err := client.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
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

	queryParams.Add("model", model)
	queryParams.Add("width", strconv.Itoa(params.Width))
	queryParams.Add("height", strconv.Itoa(params.Height))
	queryParams.Add("nologo", "true")
	queryParams.Add("safe", "false")
	queryParams.Add("nsfw", "true")
	queryParams.Add("isChild", "false")
	queryParams.Add("seed", seed)

	urlObj, err := url_package.Parse(link)
	if err != nil {
		fmt.Println("Error parsing URL:", err)
		os.Exit(1)
	}

	urlObj.RawQuery = queryParams.Encode()

	req, _ := http.NewRequest("GET", urlObj.String(), nil)

	res, err := client.Do(req)

	if err != nil {
		fmt.Fprint(os.Stderr, err)
		os.Exit(1)
	}

	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		responseText := string(body)

		fmt.Fprintf(os.Stderr, "Some error has occurred. Try again (perhaps with a different model).\nError: %v", responseText)
		os.Exit(1)
	}

	file, err := os.Create(filepath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}
	defer file.Close()

	// Copy the response body (image data) to the file
	_, err = io.Copy(file, res.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v", err)
		os.Exit(1)
	}

	return filepath
}
