package imagegen

import (
	"fmt"
	"github.com/aandrew-me/tgpt/v2/src/imagegen/arta"
	pollinations_img "github.com/aandrew-me/tgpt/v2/src/imagegen/pollinations"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	"github.com/fatih/color"
)

var bold = color.New(color.Bold)

func GenerateImg(prompt string, params structs.ImageParams, isQuite bool) {
	switch params.Provider {
	case "pollinations", "":
		if !isQuite {
			bold.Println("Generating image with pollinations.ai...")
		}
		filename := pollinations_img.GenerateImagePollinations(prompt, params)
		if !isQuite {
			fmt.Printf("Saved image as %v\n", filename)
		} else {
			fmt.Println(filename)
		}

	case "arta":
		if !isQuite {
			bold.Println("Generating image with arta...")
		}
		arta.Main(prompt, params, isQuite)
	default:
		utils.PrintError("Such a provider doesn't exist")

		return
	}
}
