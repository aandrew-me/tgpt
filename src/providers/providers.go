package providers

import (
	"fmt"
	"os"

	"github.com/aandrew-me/tgpt/v2/src/providers/anyapi"
	"github.com/aandrew-me/tgpt/v2/src/providers/deepseek"
	"github.com/aandrew-me/tgpt/v2/src/providers/gemini"
	"github.com/aandrew-me/tgpt/v2/src/providers/groq"
	"github.com/aandrew-me/tgpt/v2/src/providers/isou"
	"github.com/aandrew-me/tgpt/v2/src/providers/kimi"
	"github.com/aandrew-me/tgpt/v2/src/providers/koboldai"
	"github.com/aandrew-me/tgpt/v2/src/providers/litellm"
	"github.com/aandrew-me/tgpt/v2/src/providers/minimax"
	"github.com/aandrew-me/tgpt/v2/src/providers/ollama"
	"github.com/aandrew-me/tgpt/v2/src/providers/opencode"
	"github.com/aandrew-me/tgpt/v2/src/providers/openai"
	"github.com/aandrew-me/tgpt/v2/src/providers/pollinations"
	"github.com/aandrew-me/tgpt/v2/src/providers/powerbrain"
	"github.com/aandrew-me/tgpt/v2/src/providers/sky"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	http "github.com/bogdanfinn/fhttp"
)

var AvailableProviders = []string{
	"", "anyapi", "deepseek", "isou", "gemini", "groq", "kimi", "koboldai", "litellm", "minimax", "ollama", "ollamacloud", "opencode", "openai", "pollinations", "powerbrain", "sky",
}

func IsValidProvider(name string) bool {
	for _, ap := range AvailableProviders {
		if name == ap {
			return true
		}
	}
	return false
}

func GetMainText(line string, provider string, input string) string {
	switch provider {
	case "anyapi":
		return anyapi.GetMainText(line)
	case "deepseek":
		return deepseek.GetMainText(line)
	case "isou":
		return isou.GetMainText((line))
	case "gemini":
		return gemini.GetMainText(line)
	case "groq":
		return groq.GetMainText(line)
	case "kimi":
		return kimi.GetMainText(line)
	case "koboldai":
		return koboldai.GetMainText(line)
	case "litellm":
		return litellm.GetMainText(line)
	case "minimax":
		return minimax.GetMainText(line)
	case "ollama":
		return ollama.GetMainText(line)
	case "ollamacloud":
		return ollama.GetCloudMainText(line)
	case "opencode":
		return opencode.GetMainText(line)
	case "openai":
		return openai.GetMainText(line)
	case "pollinations":
		return pollinations.GetMainText(line)
	case "powerbrain":
		return powerbrain.GetMainText(line)
	case "sky":
		return sky.GetMainText(line)
	default:
		return pollinations.GetMainText(line)
	}
}

func NewRequest(input string, params structs.Params, extraOptions structs.ExtraOptions) (*http.Response, error) {
	if !IsValidProvider(params.Provider) {
		fmt.Fprintln(os.Stderr, "Invalid provider")
		os.Exit(1)
	}

	switch params.Provider {
	case "anyapi":
		return anyapi.NewRequest(input, params)
	case "deepseek":
		return deepseek.NewRequest(input, params)
	case "gemini":
		return gemini.NewRequest(input, params)
	case "groq":
		return groq.NewRequest(input, params)
	case "isou":
		return isou.NewRequest(input, params)
	case "kimi":
		return kimi.NewRequest(input, params)
	case "koboldai":
		return koboldai.NewRequest(input, params)
	case "litellm":
		return litellm.NewRequest(input, params)
	case "minimax":
		return minimax.NewRequest(input, params)
	case "ollama":
		return ollama.NewRequest(input, params)
	case "ollamacloud":
		return ollama.NewCloudRequest(input, params)
	case "opencode":
		return opencode.NewRequest(input, params)
	case "openai":
		return openai.NewRequest(input, params)
	case "pollinations":
		return pollinations.NewRequest(input, params)
	case "powerbrain":
		return powerbrain.NewRequest(input, params)
	case "sky":
		return sky.NewRequest(input, params)
	default:
		return pollinations.NewRequest(input, params)
	}

}
