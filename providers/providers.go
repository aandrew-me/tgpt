package providers

import (
	"github.com/aandrew-me/tgpt/v2/providers/fakeopen"
	"github.com/aandrew-me/tgpt/v2/providers/koboldai"
	"github.com/aandrew-me/tgpt/v2/providers/openai"
	"github.com/aandrew-me/tgpt/v2/providers/opengpts"
	"github.com/aandrew-me/tgpt/v2/providers/phind"
	"github.com/aandrew-me/tgpt/v2/structs"
	http "github.com/bogdanfinn/fhttp"
)

func GetMainText(line string, provider string) string {
	if provider == "fakeopen" {
		return fakeopen.GetMainText(line)
	} else if provider == "openai" {
		return openai.GetMainText(line)
	} else if provider == "opengpts" {
		return opengpts.GetMainText(line)
	} else if provider == "koboldai" {
		return koboldai.GetMainText(line)
	} else if provider == "phind" {
		return phind.GetMainText(line)
	}

	return opengpts.GetMainText(line)
}

func NewRequest(input string, params structs.Params, prevMessages string) (*http.Response, error) {
	if params.Provider == "fakeopen" {
		return fakeopen.NewRequest(input, params, prevMessages)
	} else if params.Provider == "openai" {
		return openai.NewRequest(input, params, prevMessages)
	} else if params.Provider == "opengpts" {
		return opengpts.NewRequest(input, params, prevMessages)
	} else if params.Provider == "koboldai" {
		return koboldai.NewRequest(input, params, "")
	} else if params.Provider == "phind" {
		return phind.NewRequest(input, params, "")
	}

	return opengpts.NewRequest(input, params, prevMessages)
}
