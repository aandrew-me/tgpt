package providers

import (
	"fmt"
	"os"

	"github.com/aandrew-me/tgpt/v2/providers/blackboxai"
	"github.com/aandrew-me/tgpt/v2/providers/groq"
	"github.com/aandrew-me/tgpt/v2/providers/koboldai"
	"github.com/aandrew-me/tgpt/v2/providers/ollama"
	"github.com/aandrew-me/tgpt/v2/providers/openai"
	"github.com/aandrew-me/tgpt/v2/providers/opengpts"
	"github.com/aandrew-me/tgpt/v2/providers/phind"
	"github.com/aandrew-me/tgpt/v2/structs"
	http "github.com/bogdanfinn/fhttp"
)

var availableProviders = []string{
	"", "opengpts", "ollama", "openai", "phind", "koboldai", "blackboxai", "groq",
}

func GetMainText(line string, provider string, input string) string {
	switch provider {
	case "blackboxai":
		return blackboxai.GetMainText(line)
	case "groq":
		return groq.GetMainText(line)
	case "koboldai":
		return koboldai.GetMainText(line)
	case "ollama":
		return ollama.GetMainText(line)
	case "opengpts":
		return opengpts.GetMainText(line, input)
	case "openai":
		return openai.GetMainText(line)
	default:
		return phind.GetMainText(line)
	}
}

func NewRequest(input string, params structs.Params, extraOptions structs.ExtraOptions) (*http.Response, error) {
	validProvider := false
	for _, str := range availableProviders {
		if str == params.Provider {
			validProvider = true
			break
		}
	}
	if !validProvider {
		fmt.Fprintln(os.Stderr, "Invalid provider")
		os.Exit(1)
	}

	switch params.Provider {
	case "blackboxai":
		return blackboxai.NewRequest(input, params)
	case "groq":
		return groq.NewRequest(input, params)
	case "koboldai":
		return koboldai.NewRequest(input, params)
	case "ollama":
		return ollama.NewRequest(input, params)
	case "opengpts":
		return opengpts.NewRequest(input, params)
	case "openai":
		return openai.NewRequest(input, params)
	default:
		return phind.NewRequest(input, params)
	}

}
