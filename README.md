<p align="center"><img src="tgpt.svg"></p>

# Terminal GPT (tgpt) 🚀

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt/releases/latest)
[![AUR version](https://img.shields.io/aur/version/tgpt-bin?label=AUR%3A%20tgpt-bin)](https://aur.archlinux.org/packages/tgpt-bin)

tgpt is a cross-platform command-line interface (CLI) tool that allows you to use AI chatbot in your Terminal without requiring API keys. 

### Currently available providers: 
- [OpenGPTs](https://opengpts-example-vz4y4ooboq-uc.a.run.app/) (GPT-3.5-turbo)
- [KoboldAI](https://koboldai-koboldcpp-tiefighter.hf.space/)  (koboldcpp/HF_SPACE_Tiefighter-13B)
- [Phind](https://www.phind.com/agent) (Phind Model)
<!-- - [Llama2](https://www.llama2.ai/) (Llama 2 70b) -->
- [Blackbox AI](https://www.blackbox.ai/) (Blackbox model)
- [OpenAI](https://platform.openai.com/docs/guides/text-generation/chat-completions-api) (All models, Requires API Key)

**Image Generation Model**: Craiyon V3

## Usage 💬

```
Usage: tgpt [Flags] [Prompt]

Flags:
-s, --shell                                        Generate and Execute shell commands. (Experimental) 
-c, --code                                         Generate Code. (Experimental)
-q, --quiet                                        Gives response back without loading animation
-w, --whole                                        Gives response back as a whole text
-img, --image                                      Generate images from text
--provider                                         Set Provider. Detailed information has been provided below (Env: AI_PROVIDER)

Some additional options can be set. However not all options are supported by all providers. Not supported options will just be ignored.
--model                                            Set Model
--key                                              Set API Key
--url                                              Set OpenAI API endpoint url
--temperature                                      Set temperature
--top_p                                            Set top_p
--max_length                                       Set max response length

Options:
-v, --version                                      Print version 
-h, --help                                         Print help message 
-i, --interactive                                  Start normal interactive mode 
-m, --multiline                                    Start multi-line interactive mode 
-cl, --changelog                                   See changelog of versions 
-u, --update                                       Update program 

Providers:
The default provider is phind. The AI_PROVIDER environment variable can be used to specify a different provider.
Available providers to use: openai, opengpts, koboldai, phind, blackboxai, ollama and groq

Provider: openai
Needs API key to work and supports various models. Recognizes the OPENAI_API_KEY and OPENAI_MODEL environment variables.

Provider: opengpts
Uses gpt-3.5-turbo only. Do not use with sensitive data

Provider: koboldai
Uses koboldcpp/HF_SPACE_Tiefighter-13B only, answers from novels

Provider: phind
Uses Phind Model. Great for developers

Provider: blackboxai
Uses BlackBox model. Great for developers

Provider: ollama
Needs to be run locally. Supports many models

Provider: groq
Requires a free API Key. Supports LLaMA2-70b & Mixtral-8x7b

Examples:
tgpt "What is internet?"
tgpt -m
tgpt -s "How to update my system?"
tgpt --provider opengpts "What is 1+1"
tgpt --provider openai --key "sk-xxxx" --model "gpt-3.5-turbo" "What is 1+1"
cat install.sh | tgpt "Explain the code"
```

![demo](https://user-images.githubusercontent.com/66430340/233759296-c4cf8cf2-0cab-48aa-9e84-40765b823282.gif)

## Installation ⏬

### Download for GNU/Linux 🐧 or MacOS 🍎

The default download location is `/usr/local/bin`, but you can change it in the command to use a different location. However, make sure the location is added to your PATH environment variable for easy accessibility.

You can download it with the following command:

```bash
curl -sSL https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash -s /usr/local/bin
```

If you are using Arch Linux, you can install the [AUR package](https://aur.archlinux.org/packages/tgpt-bin) with `paru`:

```bash
paru -S tgpt-bin
```

Or with `yay`:

```bash
yay -S tgpt-bin
```

### FreeBSD 😈 

Currently, the port is not yet in the quarterly branch of the FreeBSD ports tree.

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

<!-- -   **PowerShell:** Open PowerShell as administrator and run the following command:
    
    ```bash
    Invoke-WebRequest https://raw.githubusercontent.com/aandrew-me/tgpt/main/install-win.ps1 -OutFile "$PWD\install-win.ps1";  .\install-win.ps1
    ```

    If you receive an error stating "execution of scripts is disabled on this system," run this command instead (and confirm with a "Y"):

    ```bash
    Set-ExecutionPolicy -ExecutionPolicy RemoteSigned; Invoke-WebRequest https://raw.githubusercontent.com/aandrew-me/tgpt/main/install-win.ps1 -OutFile "$PWD\install-win.ps1";  .\install-win.ps1
    ``` -->

## Updating ⬆️
If you installed the program with the installation script, you may update it with
```bash
tgpt -u
```
**It may require admin privileges.**
### Proxy

Support:
- Http Proxy [ `http://ip:port` ]
- Http Auth [ `http://user:pass@ip:port` ]
- Socks5 Proxy [ `socks5://ip:port ]`
- Socks5 Auth [ `socks5://user:pass@ip:port` ]

If you want to use a proxy, create `proxy.txt` file in the same directory from where you are executing the file and write your proxy configuration there.

Example:

```bash
http://127.0.0.1:8080
```

### From Release

You can download the executable for your operating system, rename it to `tgpt` (or any other desired name), and then execute it by typing `./tgpt` while in that directory. Alternatively, you can add it to your PATH environmental variable and then execute it by simply typing `tgpt`.


## Uninstalling
If you installed with the install script, you can execute the following command to remove the tgpt executable
```
sudo rm $(which tgpt)
```
Configuration file is usually located in `~/.config/tgpt` on GNU/Linux Systems and in `"Library/Application Support/tgpt"` on MacOS
