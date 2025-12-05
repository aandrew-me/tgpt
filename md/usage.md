## Usage 

`tgpt [Flags] [Prompt]`


### Flags:

```
-s, --shell               Generate and Execute shell commands. (Experimental) 
-c, --code                Generate Code. (Experimental)
-q, --quiet               Gives response back without loading animation
-w, --whole               Gives response back as a whole text
-img, --image             Generate images from text
--provider                Set Provider. Detailed information has been provided below. (Env: AI_PROVIDER)
```

### Some additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.

```
--model                   Set Model
--key                     Set API Key. (Env: AI_API_KEY)
--url                     Set OpenAI API endpoint url
--temperature             Set temperature
--top_p                   Set top_p
--log                     Set filepath to log conversation to (For interactive modes)  
--preprompt               Set preprompt
-y                        Execute shell command without confirmation
```

### Options supported for image generation (with -image flag)

```
--out                     Output image filename (Supported by pollinations)
--height                  Output image height (Supported by pollinations)
--width                   Output image width (Supported by pollinations)
--img_count               Output image count (Supported by arta)
--img_negative            Negative prompt (Supported by arta)
--img_ratio               Output image ratio (Supported by arta, some models may not support it)
```

### General Options:

```
-v, --version             Print version
-h, --help                Print help message
-i, --interactive         Start normal interactive mode
-m, --multiline           Start multi-line interactive mode
-is, --interactive-shell  Start interactive shell mode
-cl, --changelog          See changelog of versions
```

## Providers

The default provider is `phind`. 

The `AI_PROVIDER` environment variable can be used to specify a different provider.

Available providers to use: `deepseek`, `duckduckgo`, `gemini`, `groq`, `isou`, `koboldai`, `ollama`, `openai`, `phind`, and `pollinations`      

### deepseek
Uses `deepseek-reasoner` model by default. 

Requires API key. 

Recognizes the `DEEPSEEK_API_KEY` and `DEEPSEEK_MODEL` environment variables

### duckduckgo
Free provider, no API key required.

Uses `o3-mini` model by default. Other supported models: `meta-llama/Meta-Llama-3.1-70B-Instruct-Turbo`, `mistralai/Mixtral-8x7B-Instruct-v0.1`, `claude-3-haiku-20240307`

### groq
Requires a free API Key. 

Supported models: [https://console.groq.com/docs/models](https://console.groq.com/docs/models)

### gemini
Requires a free API key. 

Supported models: [https://ai.google.dev/gemini-api/docs/models](https://ai.google.dev/gemini-api/docs/models). 

Recognizes the `GEMINI_API_KEY` and `GEMINI_MODEL` environment variables

### isou

Free provider with web search

### koboldai
Uses `koboldcpp/HF_SPACE_Tiefighter-13B` only, answers from novels

### ollama
Ollama needs to be run locally. Supports many models

### openai
Needs API key to work and supports various models. 

Recognizes the `OPENAI_API_KEY` and `OPENAI_MODEL` environment variables. Supports custom urls with --url

### phind
Uses Phind Model. Great for developers

### pollinations

Supported models: [https://text.pollinations.ai/models](https://text.pollinations.ai/models)

Completely free, default model is `gpt-4o`. 

### Environment Variables

The following environment variables can be used to configure tgpt:

| Variable           | Description                              |
|--------------------|------------------------------------------|
| `AI_PROVIDER`        | Default provider (e.g., openai, gemini)  |
| `AI_API_KEY`         | Fallback API key (used by OpenAI if provider-specific key not set) |
| `OPENAI_API_KEY`     | API key for OpenAI provider              |
| `OPENAI_MODEL`       | Model for OpenAI provider                |
| `OPENAI_URL`         | Custom endpoint URL for OpenAI           |
| `GEMINI_API_KEY`     | API key for Gemini provider              |
| `GEMINI_MODEL`       | Model for Gemini provider                |
| `DEEPSEEK_API_KEY`   | API key for DeepSeek provider            |

Note: Command line flags (`--key`, `--model`, `--url`) override environment variables when both are set.

## Image Generation Providers

### pollinations
Supported models: `flux`, `turbo`

### arta
Supported models: `F Dev`, `Minimalistic Logo`, `F Retro Anime`, `Low Poly`, `F Super Realism`, `F Realism`, `Embroidery tattoo`, `Old school colored`, `Hand-drawn Logo`, `GPT4o Ghibli`, `F Pencil`, `F Retrocomic`, `Juggernaut-xl`, `Medieval`, `F Softserve`, `No Style`, `New School`, `Trash Polka`, `Anime tattoo`, `F Jojoso`, `Grunge Logo`, `F Dreamscape`, `F Whimscape`, `Kawaii`, `Flame design`, `Old School`, `Katayama-mix-xl`, `On limbs black`, `SDXL L`, `F Pixel`, `Realistic tattoo`, `Flux`, `Graffiti`, `F Anime Journey`, `F Koda`, `Gradient Logo`, `On limbs color`, `Elegant Logo`, `Random Text`, `F Face Realism`, `Playground-xl`, `Epic Logo`, `Photographic`, `Mascots Logo`, `Surrealism`, `Monogram Logo`, `Chicano`, `Pony-xl`, `Anima-pencil-xl`, `Mini tattoo`, `Dotwork`, `F Miniature`, `Watercolor`, `Futuristic Logo`, `RevAnimated`, `Geometric Logo`, `Emblem Logo`, `Biomech`, `Combination Logo`, `Death metal`, `F Dalle Mix`, `Neo-traditional`, `Cheyenne-xl`, `Realistic-stock-xl`, `F Epic Realism`, `Anything-xl`, `Japanese_2`, `F Pro`, `GPT4o`, `Black Ink`, `F Midjorney`, `Abstract Logo`, `3D Logo`, `Red and Black`, `High GPT4o`, `Dreamshaper-xl`, `Yamers-realistic-xl`, `Cor-epica-xl`, `F Anime`, `F Real Anime`, `Professional`, `Fantasy Art`, `Cinematic Art`, `Vincent Van Gogh`, `SDXL 1.0`

Supported ratios:
`1:1`, `2:3`, `3:2`, `3:4`, `4:3`, `9:16`, `16:9`, `9:21`, `21:9`

## Examples

```tgpt "What is internet?"
tgpt -m
tgpt -s "How to update my system?"
tgpt --provider duckduckgo "What is 1+1"
tgpt --img "cat"
tgpt --img --out ~/my-cat.jpg --height 256 --width 256 "cat"
tgpt --provider openai --key "sk-xxxx" --model "gpt-3.5-turbo" "What is 1+1"
cat install.sh | tgpt "Explain the code"
```