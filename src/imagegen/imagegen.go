package imagegen

import (
	"fmt"
	"github.com/aandrew-me/tgpt/v2/src/imagegen/aihorde"
	"github.com/aandrew-me/tgpt/v2/src/imagegen/anyapi"
	"github.com/aandrew-me/tgpt/v2/src/imagegen/magicstudio"
	pollinations_img "github.com/aandrew-me/tgpt/v2/src/imagegen/pollinations"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	"github.com/aandrew-me/tgpt/v2/src/utils"
	"github.com/fatih/color"
)

var bold = color.New(color.Bold)

func GenerateImg(prompt string, params structs.ImageParams, isQuite bool) {
	switch params.Provider {
	case "aihorde":
		if !isQuite {
			bold.Println("Generating image with AI-Horde (stablehorde.net)...")
		}
		filename := aihorde.GenerateImage(prompt, params)
		if !isQuite {
			fmt.Printf("Saved image as %v\n", filename)
			_ = utils.OpenImage(filename)
		} else {
			fmt.Println(filename)
		}

	case "pollinations":
		if !isQuite {
			bold.Println("Generating image with pollinations.ai...")
		}
		filename := pollinations_img.GenerateImagePollinations(prompt, params)
		if !isQuite {
			fmt.Printf("Saved image as %v\n", filename)
			_ = utils.OpenImage(filename)
		} else {
			fmt.Println(filename)
		}

	case "anyapi":
		if !isQuite {
			bold.Println("Generating image with anyapi.ai...")
		}
		filename := anyapi.GenerateImage(prompt, params)
		if !isQuite {
			fmt.Printf("Saved image as %v\n", filename)
			_ = utils.OpenImage(filename)
		} else {
			fmt.Println(filename)
		}
	
	case "magicstudio", "":
		if !isQuite {
			bold.Println("Generating image with magicstudio...")
		}
		filename := magicstudio.GenerateImageMagicStudio(prompt, params)
		if !isQuite {
			fmt.Printf("Saved image as %v\n", filename)
			_ = utils.OpenImage(filename)
		} else {
			fmt.Println(filename)
		}

	default:
		utils.PrintError("Such a provider doesn't exist")

		return
	}
}
