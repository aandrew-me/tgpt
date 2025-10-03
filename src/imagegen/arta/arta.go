package arta

// From https://github.com/rohitaryal/imazes

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"strings"
	"time"

	"github.com/aandrew-me/tgpt/v2/src/client"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	http "github.com/bogdanfinn/fhttp"
)

type Image struct {
	Prompt   string
	Negative string
	Style    string
	Count    string
	Steps    string
	Ratio    string
}

type TokenResponse struct {
	Kind         string `json:"kind"`
	IdToken      string `json:"idToken"`
	RefreshToken string `json:"refreshToken"`
	ExpiresIn    string `json:"expiresIn"`
	LocalId      string `json:"localId"`
}

type StatusResponse struct {
	RecordID     string          `json:"record_id"`
	Status       string          `json:"status"`
	Response     []ImageResponse `json:"response"`
	ErrorCode    any             `json:"error_code"`
	ErrorDetails any             `json:"error_details"`
	Seed         int             `json:"seed"`
	Detail       []Error         `json:"detail"`
}

type Error struct {
	Msg string `json:"msg"`
}

type ImageResponse struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	IsBlur bool   `json:"isBlur"`
	MIME   string `json:"MIME"`
}

var Styles = []string{
	"F Dev",
	"Minimalistic Logo",
	"F Retro Anime",
	"Low Poly",
	"F Super Realism",
	"F Realism",
	"Embroidery tattoo",
	"Old school colored",
	"Hand-drawn Logo",
	"GPT4o Ghibli",
	"F Pencil",
	"F Retrocomic",
	"Juggernaut-xl",
	"Medieval",
	"F Softserve",
	"No Style",
	"New School",
	"Trash Polka",
	"Anime tattoo",
	"F Jojoso",
	"Grunge Logo",
	"F Dreamscape",
	"F Whimscape",
	"Kawaii",
	"Flame design",
	"Old School",
	"Katayama-mix-xl",
	"On limbs black",
	"SDXL L",
	"F Pixel",
	"Realistic tattoo",
	"Flux",
	"Graffiti",
	"F Anime Journey",
	"F Koda",
	"Gradient Logo",
	"On limbs color",
	"Elegant Logo",
	"Random Text",
	"F Face Realism",
	"Playground-xl",
	"Epic Logo",
	"Photographic",
	"Mascots Logo",
	"Surrealism",
	"Monogram Logo",
	"Chicano",
	"Pony-xl",
	"Anima-pencil-xl",
	"Mini tattoo",
	"Dotwork",
	"F Miniature",
	"Watercolor",
	"Futuristic Logo",
	"RevAnimated",
	"Geometric Logo",
	"Emblem Logo",
	"Biomech",
	"Combination Logo",
	"Death metal",
	"F Dalle Mix",
	"Neo-traditional",
	"Cheyenne-xl",
	"Realistic-stock-xl",
	"F Epic Realism",
	"Anything-xl",
	"Japanese_2",
	"F Pro",
	"GPT4o",
	"Black Ink",
	"F Midjorney",
	"Abstract Logo",
	"3D Logo",
	"Red and Black",
	"High GPT4o",
	"Dreamshaper-xl",
	"Yamers-realistic-xl",
	"Cor-epica-xl",
	"F Anime",
	"F Real Anime",
	"Professional",
	"Fantasy Art",
	"Cinematic Art",
	"Vincent Van Gogh",
	"SDXL 1.0",
}

var Ratios = []string{"1:1", "2:3", "3:2", "3:4", "4:3", "9:16", "16:9", "9:21", "21:9"}

func PrintModels() {
	fmt.Println("Supported models: ")

	for i, style := range Styles {
		if i == len(Styles)-1 {
			fmt.Printf("%v\n\n", style)
		} else {
			fmt.Print(style + ", ")

		}
	}
}

func PrintRatios() {
	fmt.Println("Supported ratios: ")

	for i, ratio := range Ratios {
		if i == len(Ratios)-1 {
			fmt.Printf("%v\n", ratio)
		} else {
			fmt.Print(ratio + ", ")

		}
	}
}

// Generates an authentication token that is required by each client to generate images
func GenerateToken() *TokenResponse {
	// URL to send POST req to obtain the token
	url := "https://www.googleapis.com/identitytoolkit/v3/relyingparty/signupNewUser?key=AIzaSyB3-71wG0fIt0shj0ee4fvx1shcjJHGrrQ"
	jsonBody := []byte(`{"clientType":"CLIENT_TYPE_ANDROID"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		panic(err)
	}

	// Set some important headers
	req.Header.Set("X-Android-Cert", "ADC09FCA89A2CE4D0D139031A2A587FA87EE4155")
	req.Header.Set("X-Firebase-Gmpid", "1:713239656559:android:f9e37753e9ee7324cb759a")
	req.Header.Set("X-Firebase-Client", "H4sIAAAAAAAA_6tWykhNLCpJSk0sKVayio7VUSpLLSrOzM9TslIyUqoFAFyivEQfAAAA")
	req.Header.Set("X-Client-Version", "Android/Fallback/X22003001/FirebaseCore-Android")
	req.Header.Set("User-Agent", "Dalvik/2.1.0 (Linux; U; Android 15;)")
	req.Header.Set("X-Android-Package", "ai.generated.art.maker.image.picture.photo.generator.painting")

	// Make request
	client, err := client.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	// Parse JSON to TokenResponse format
	var data TokenResponse
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &data); err != nil {
		panic(err)
	}

	return &data
}

// Sends request to generate image with a prompt
// but will respond with a status_id which is an identifier
// for specific request which when queried determines the
// current status of generation (like queued, completed,)
// in the server
func GenerateImage(imageDescription Image, token string, debug bool) *StatusResponse {
	url := "https://img-gen-prod.ai-arta.com/api/v1/text2image"

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	writer.WriteField("prompt", imageDescription.Prompt)
	writer.WriteField("negative_prompt", imageDescription.Negative)
	writer.WriteField("style", imageDescription.Style)
	writer.WriteField("images_num", imageDescription.Count)
	writer.WriteField("cfg_scale", "7")
	writer.WriteField("steps", imageDescription.Steps)
	writer.WriteField("aspect_ratio", imageDescription.Ratio)

	if err := writer.Close(); err != nil {
		panic(err)
	}

	writer.Close()

	req, err := http.NewRequest("POST", url, &body)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", token)
	req.Header.Set("User-Agent", "AiArt/4.18.6 okHttp/4.12.0 Android R")

	client, err := client.NewClient()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}

	defer resp.Body.Close()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	var data StatusResponse

	if err := json.Unmarshal(responseBody, &data); err != nil {
		panic(err)
	}

	if resp.StatusCode != 200 {
		if len(data.Detail) > 0 {
			utils.PrintError("Error: " + data.Detail[0].Msg)
		} else {
			utils.PrintError(string(responseBody))
		}

		os.Exit(1)
	}

	return &data
}

func GetImage(statusId, token string) *StatusResponse {
	url := "https://img-gen-prod.ai-arta.com/api/v1/text2image/" + statusId + "/status"
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", token)
	req.Header.Add("User-Agent", "AiArt/3.23.12 okHttp/4.12.0 Android VANILLA_ICE_CREAM")

	client, err := client.NewClient()

	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	resp, err := client.Do(req)

	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var data StatusResponse
	var body []byte

	if body, err = io.ReadAll(resp.Body); err != nil {
		panic(err)
	}

	if err := json.Unmarshal(body, &data); err != nil {
		panic(err)
	}

	return &data
}

func Main(prompt string, params structs.ImageParams, isQuite bool) {
	model := "SDXL 1.0"

	if params.ApiModel != "" {
		model = params.ApiModel
	}

	imageDescription := Image{
		Prompt:   prompt,
		Negative: params.ImgNegativePrompt,
		Style:    model,
		Count:    params.ImgCount,
		Steps:    "40",
		Ratio:    params.ImgRatio,
	}

	token := GenerateToken().IdToken

	generator := GenerateImage(imageDescription, token, true)

	for {
		status := GetImage(generator.RecordID, token)

		if status.Status != "DONE" {
			time.Sleep(5 * time.Second)
			continue
		} else {
			var images = status.Response
			for i, image := range images {
				fmt.Printf("%v.Image URL: %v\n", i+1, image.URL)
			}

			if !isQuite {
				fmt.Print("\nSave images? [y/n]: ")
				reader := bufio.NewReader(os.Stdin)
				userInput, _ := reader.ReadString('\n')
				userInput = strings.TrimSpace(userInput)

				if userInput == "y" || userInput == "" {
					for i := range images {
						utils.DownloadImage(images[i].URL, ".")
					}
				}
			}

			return
		}
	}

}
