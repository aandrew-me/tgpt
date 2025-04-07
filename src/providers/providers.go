package providers

import (
	"fmt"
	"os"

	"github.com/aandrew-me/tgpt/v2/src/providers/blackboxai"
	"github.com/aandrew-me/tgpt/v2/src/providers/deepseek"
	"github.com/aandrew-me/tgpt/v2/src/providers/duckduckgo"
	"github.com/aandrew-me/tgpt/v2/src/providers/gemini"
	"github.com/aandrew-me/tgpt/v2/src/providers/groq"
	"github.com/aandrew-me/tgpt/v2/src/providers/isou"
	"github.com/aandrew-me/tgpt/v2/src/providers/koboldai"
	"github.com/aandrew-me/tgpt/v2/src/providers/ollama"
	"github.com/aandrew-me/tgpt/v2/src/providers/openai"
	"github.com/aandrew-me/tgpt/v2/src/providers/phind"
	"github.com/aandrew-me/tgpt/v2/src/providers/pollinations"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	http "github.com/bogdanfinn/fhttp"
)

var availableProviders = []string{
	"", "blackboxai", "deepseek", "duckduckgo", "isou", "groq", "koboldai", "ollama", "openai", "phind", "pollinations", "gemini",
}

func GetMainText(line string, provider string, input string) string {
	switch provider {
	case "blackboxai":
		return blackboxai.GetMainText(line)
	case "deepseek":
		return deepseek.GetMainText(line)
	case "duckduckgo":
		return duckduckgo.GetMainText(line)
	case "isou":
		return isou.GetMainText((line))
	case "groq":
		return groq.GetMainText(line)
	case "koboldai":
		return koboldai.GetMainText(line)
	case "ollama":
		return ollama.GetMainText(line)
	case "openai":
		return openai.GetMainText(line)
	case "pollinations":
		return pollinations.GetMainText(line)
	case "gemini":
		return gemini.GetMainText(line)
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
	case "deepseek":
		return deepseek.NewRequest(input, params)
	case "duckduckgo":
		return duckduckgo.NewRequest(input, params, params.PrevMessages)
	case "groq":
		return groq.NewRequest(input, params)
	case "isou":
		return isou.NewRequest(input, params)
	case "koboldai":
		return koboldai.NewRequest(input, params)
	case "ollama":
		return ollama.NewRequest(input, params)
	case "openai":
		return openai.NewRequest(input, params)
	case "pollinations":
		return pollinations.NewRequest(input, params)
	case "gemini":
		return gemini.NewRequest(input, params)
	default:
		return phind.NewRequest(input, params)
	}

}
