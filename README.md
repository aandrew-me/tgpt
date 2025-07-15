<p align="center"><img src="tgpt.svg"></p>

# Terminal GPT (tgpt) 🚀

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt/releases/latest)
![Arch Linux package](https://img.shields.io/archlinux/v/extra/x86_64/tgpt)
![Chocolatey Version](https://img.shields.io/chocolatey/v/tgpt)

**tgpt** is a Cross-platform Command-Line Interface (CLI) tool that allows you to use AI in your Terminal.

<img src="https://github.com/user-attachments/assets/1b554b99-79ca-45b7-87ff-7713b7fd9437" alt="Demo" width="500" height="330">


### Currently available providers: 
- [Deepseek](https://www.deepseek.com/) (Requires API key)
- [Groq](https://groq.com/) (Requires a free API Key. [Many models](https://console.groq.com/docs/models))
- [Isou](https://isou.chat/) (Free) (Deepseek-chat with SEARXNG)
- [KoboldAI](https://koboldai-koboldcpp-tiefighter.hf.space/) (Free) (koboldcpp/HF_SPACE_Tiefighter-13B)
- [Ollama](https://www.ollama.com/) (Local models) (Supports many models)
- [OpenAI](https://platform.openai.com/docs/guides/text-generation/chat-completions-api) (All models, Requires API Key, supports custom endpoints)
- [Phind](https://www.phind.com/agent) (Free) (Phind Model)
- [Pollinations](https://pollinations.ai/) ([Many free models](https://text.pollinations.ai/models))
- [Gemini](https://gemini.google.com) (Require a free API keys, supports [many models](https://ai.google.dev/gemini-api/docs/models/gemini), default model `gemini-2.0-flash`)

**Image Generation Models**: 
- Arta (Free)
- Pollinations (Free) ([Models](https://image.pollinations.ai/models))

## Installation ⏬

### Download for GNU/Linux 🐧 or MacOS 🍎

The default download location is `/usr/local/bin`, but you can change it in the command to use a different location. However, make sure the location is added to your PATH environment variable for easy accessibility.

You can download it with the following command:

```bash
curl -sSL https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash -s /usr/local/bin
```

If you are using Arch Linux, you can install with pacman:

```bash
pacman -S tgpt
```


### FreeBSD 😈 

To install the [port](https://www.freshports.org/www/tgpt):
```
cd /usr/ports/www/tgpt/ && make install clean
```
To install the package, run one of these commands:
```
pkg install www/tgpt
pkg install tgpt
```

### Install with Go
You need to [add the Go install directory to your system's shell path](https://go.dev/doc/tutorial/compile-install). 

```bash
go install github.com/aandrew-me/tgpt/v2@latest
```

### Windows 🪟

-   **Scoop:** Package installation with [Scoop](https://scoop.sh/) can be done using the following command:

    ```bash
    scoop install https://raw.githubusercontent.com/aandrew-me/tgpt/main/tgpt.json
    ```
- **Chocolatey** 
    ```bash
    choco install tgpt
    ```    
### From Release

You can download the executable for your operating system, rename it to `tgpt` (or any other desired name), and then execute it by typing `./tgpt` while in that directory. Alternatively, you can add it to your PATH environmental variable and then execute it by simply typing `tgpt`.

## Updating ⬆️
If you installed the program with the installation script, you may update it with
```bash
tgpt -u
```
**It may require admin privileges.**

## Usage 

```
Usage: tgpt [Flags] [Prompt]

Flags:
-s, --shell                                        Generate and Execute shell commands. (Experimental) 
-c, --code                                         Generate Code. (Experimental)
-q, --quiet                                        Gives response back without loading animation
-w, --whole                                        Gives response back as a whole text
-img, --image                                      Generate images from text
--provider                                         Set Provider. Detailed information has been provided below. (Env: AI_PROVIDER)

Some additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.
--model                                            Set Model
--key                                              Set API Key. (Env: AI_API_KEY)
--url                                              Set OpenAI API endpoint url
--temperature                                      Set temperature
--top_p                                            Set top_p
--log                                              Set filepath to log conversation to (For interactive modes)  
--preprompt                                        Set preprompt
-y                                                 Execute shell command without confirmation

Options supported for image generation (with -image flag)
--out                                              Output image filename (Supported by pollinations)
--height                                           Output image height (Supported by pollinations)
--width                                            Output image width (Supported by pollinations)
--img_count                                        Output image count (Supported by arta)
--img_negative                                     Negative prompt (Supported by arta)
--img_ratio                                        Output image ratio (Supported by arta, some models may not support it)

Options:
-v, --version                                      Print version
-h, --help                                         Print help message
-i, --interactive                                  Start normal interactive mode
-m, --multiline                                    Start multi-line interactive mode
-is, --interactive-shell                           Start interactive shell mode
-cl, --changelog                                   See changelog of versions

Providers:
The default provider is phind. The AI_PROVIDER environment variable can be used to specify a different provider.
Available providers to use: deepseek, gemini, groq, isou, koboldai, ollama, openai, pollinations and phind      

Provider: deepseek
Uses deepseek-reasoner model by default. Requires API key. Recognizes the DEEPSEEK_API_KEY and DEEPSEEK_MODEL environment variables

Provider: groq
Requires a free API Key. Supported models: https://console.groq.com/docs/models

Provider: gemini
Requires a free API key. https://aistudio.google.com/apikey

Provider: isou
Free provider with web search

Provider: koboldai
Uses koboldcpp/HF_SPACE_Tiefighter-13B only, answers from novels

Provider: ollama
Needs to be run locally. Supports many models

Provider: openai
Needs API key to work and supports various models. Recognizes the OPENAI_API_KEY and OPENAI_MODEL environment variables. Supports custom urls with --url

Provider: phind
Uses Phind Model. Great for developers

Provider: pollinations
Completely free, default model is gpt-4o. Supported models: https://text.pollinations.ai/models

Image generation providers:

Provider: pollinations
Supported models: flux, turbo

Provider: arta
Supported models:
Medieval, Vincent Van Gogh, F Dev, Low Poly, Dreamshaper-xl, Anima-pencil-xl, Biomech, Trash Polka, No Style, Cheyenne-xl, Chicano, Embroidery tattoo, Red and Black, Fantasy Art, Watercolor, Dotwork, Old school colored, Realistic tattoo, Japanese_2, Realistic-stock-xl, F Pro, RevAnimated, Katayama-mix-xl, SDXL L, Cor-epica-xl, Anime tattoo, New School, Death metal, Old School, Juggernaut-xl, Photographic, SDXL 1.0, Graffiti, Mini tattoo, Surrealism, Neo-traditional, On limbs black, Yamers-realistic-xl, Pony-xl, Playground-xl, Anything-xl, Flame design, Kawaii, Cinematic Art, Professional, Flux, Black Ink, Epicrealism-xl

Supported ratios:
1:1, 2:3, 3:2, 3:4, 4:3, 9:16, 16:9, 9:21, 21:9

Examples:
tgpt "What is internet?"
tgpt -m
tgpt -s "How to update my system?"
tgpt --provider duckduckgo "What is 1+1"
tgpt --img "cat"
tgpt --img --out ~/my-cat.jpg --height 256 --width 256 "cat"
tgpt --provider openai --key "sk-xxxx" --model "gpt-3.5-turbo" "What is 1+1"
cat install.sh | tgpt "Explain the code"
```

### Proxy

Support:

### 1. Environment variable

`http_proxy` or `HTTP_PROXY` with following available formats:

- Http Proxy [ `http://ip:port` ]
- Http Auth [ `http://user:pass@ip:port` ]
- Socks5 Proxy [ `socks5://ip:port ]`
- Socks5 Auth [ `socks5://user:pass@ip:port` ]

### 2. Configuration file

Supported file locations:

- `./proxy.txt` (in the same directory from where you are executing)
- `~/.config/tgpt/proxy.txt`

Example:

```bash
http://127.0.0.1:8080
```

## Uninstalling
If you installed with the install script, you can execute the following command to remove the tgpt executable
```
sudo rm $(which tgpt)
```
Configuration file is usually located in `~/.config/tgpt` on GNU/Linux Systems and in `"Library/Application Support/tgpt"` on MacOS
