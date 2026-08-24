package providers

import (
	"fmt"
	"os"

	"github.com/aandrew-me/tgpt/v2/src/providers/aihorde"
	"github.com/aandrew-me/tgpt/v2/src/providers/aitopia"
	"github.com/aandrew-me/tgpt/v2/src/providers/anyapi"
	"github.com/aandrew-me/tgpt/v2/src/providers/atlascloud"
	"github.com/aandrew-me/tgpt/v2/src/providers/deepseek"
	"github.com/aandrew-me/tgpt/v2/src/providers/deepseekweb"
	"github.com/aandrew-me/tgpt/v2/src/providers/fx"
	"github.com/aandrew-me/tgpt/v2/src/providers/gemini"
	"github.com/aandrew-me/tgpt/v2/src/providers/groq"
	"github.com/aandrew-me/tgpt/v2/src/providers/isou"
	"github.com/aandrew-me/tgpt/v2/src/providers/koboldai"
	"github.com/aandrew-me/tgpt/v2/src/providers/litellm"
	"github.com/aandrew-me/tgpt/v2/src/providers/minimax"
	"github.com/aandrew-me/tgpt/v2/src/providers/ollama"
	"github.com/aandrew-me/tgpt/v2/src/providers/omniroute"
	"github.com/aandrew-me/tgpt/v2/src/providers/openai"
	"github.com/aandrew-me/tgpt/v2/src/providers/opencode"
	"github.com/aandrew-me/tgpt/v2/src/providers/openrouter"
	"github.com/aandrew-me/tgpt/v2/src/providers/pollinations"
	"github.com/aandrew-me/tgpt/v2/src/providers/powerbrain"
	"github.com/aandrew-me/tgpt/v2/src/structs"
	http "github.com/bogdanfinn/fhttp"
)

var AvailableProviders = []string{
	"anyapi", "aihorde", "aitopia", "atlascloud", "deepseek", "deepseek-web", "fx", "isou", "gemini", "groq", "koboldai", "litellm", "minimax", "ollama", "ollamacloud", "omniroute", "opencode", "openai", "openrouter", "pollinations", "powerbrain",
}

func IsValidProvider(name string) bool {
	for _, ap := range AvailableProviders {
		if name == ap {
			return true
		}
	}
	return false
}

func SupportsTools(provider string) bool {
	if provider == "" {
		provider = "opencode"
	}
	switch provider {
	case "anyapi", "atlascloud", "deepseek", "gemini", "groq", "litellm", "ollama", "omniroute", "opencode", "openai", "openrouter", "pollinations":
		return true
	default:
		return false
	}
}

func GetMainText(line string, provider string, input string) string {
	switch provider {
	case "aihorde":
		return aihorde.GetMainText(line)
	case "aitopia":
		return aitopia.GetMainText(line)
	case "anyapi":
		return anyapi.GetMainText(line)
	case "atlascloud":
		return atlascloud.GetMainText(line)
	case "deepseek":
		return deepseek.GetMainText(line)
	case "deepseek-web":
		return deepseekweb.GetMainText(line)
	case "fx":
		return fx.GetMainText(line)
	case "isou":
		return isou.GetMainText((line))
	case "gemini":
		return gemini.GetMainText(line)
	case "groq":
		return groq.GetMainText(line)
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
	case "omniroute":
		return omniroute.GetMainText(line)
	case "opencode":
		return opencode.GetMainText(line)
	case "openai":
		return openai.GetMainText(line)
	case "openrouter":
		return openrouter.GetMainText(line)
	case "pollinations":
		return pollinations.GetMainText(line)
	case "powerbrain":
		return powerbrain.GetMainText(line)
	default:
		return opencode.GetMainText(line)
	}
}

func NewRequest(input string, params structs.Params, extraOptions structs.ExtraOptions) (*http.Response, error) {
	provider := params.Provider
	if provider == "" {
		provider = "opencode"
	}
	if !IsValidProvider(provider) {
		fmt.Fprintln(os.Stderr, "Invalid provider")
		os.Exit(1)
	}

	switch provider {
	case "aihorde":
		return aihorde.NewRequest(input, params)
	case "aitopia":
		return aitopia.NewRequest(input, params)
	case "anyapi":
		return anyapi.NewRequest(input, params)
	case "atlascloud":
		return atlascloud.NewRequest(input, params)
	case "deepseek":
		return deepseek.NewRequest(input, params)
	case "deepseek-web":
		return deepseekweb.NewRequest(input, params)
	case "fx":
		return fx.NewRequest(input, params)
	case "gemini":
		return gemini.NewRequest(input, params)
	case "groq":
		return groq.NewRequest(input, params)
	case "isou":
		return isou.NewRequest(input, params)
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
	case "omniroute":
		return omniroute.NewRequest(input, params)
	case "opencode":
		return opencode.NewRequest(input, params)
	case "openai":
		return openai.NewRequest(input, params)
	case "openrouter":
		return openrouter.NewRequest(input, params)
	case "pollinations":
		return pollinations.NewRequest(input, params)
	case "powerbrain":
		return powerbrain.NewRequest(input, params)
	default:
		return opencode.NewRequest(input, params)
	}

}
