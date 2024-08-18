const cp = require("child_process")
const { writeFileSync } = require("fs")
const helpTxt = cp.execSync("go run *.go -h").toString().trim()

const readmeTxt = `<p align="center"><img src="tgpt.svg"></p>

# Terminal GPT (tgpt) 🚀

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt/releases/latest)
![Arch Linux package](https://img.shields.io/archlinux/v/extra/x86_64/tgpt)
![Chocolatey Version](https://img.shields.io/chocolatey/v/tgpt)

tgpt is a cross-platform command-line interface (CLI) tool that allows you to use AI chatbot in your Terminal without requiring API keys. 

### Currently available providers: 
- [Blackbox AI](https://www.blackbox.ai/) (Blackbox model)
- [Duckduckgo](https://duckduckgo.com/aichat) (Supports several models)
- [Groq](https://groq.com/) (Requires a free API Key. LLaMA2-70b & Mixtral-8x7b)
- [KoboldAI](https://koboldai-koboldcpp-tiefighter.hf.space/)  (koboldcpp/HF_SPACE_Tiefighter-13B)
- [Ollama](https://www.ollama.com/) (Supports many models)
- [OpenAI](https://platform.openai.com/docs/guides/text-generation/chat-completions-api) (All models, Requires API Key, supports custom endpoints)
- [Phind](https://www.phind.com/agent) (Phind Model)

**Image Generation Model**: Craiyon V3

## Usage 

\`\`\`
${helpTxt}
\`\`\`

![demo](https://user-images.githubusercontent.com/66430340/233759296-c4cf8cf2-0cab-48aa-9e84-40765b823282.gif)

## Installation ⏬

### Download for GNU/Linux 🐧 or MacOS 🍎

The default download location is \`/usr/local/bin\`, but you can change it in the command to use a different location. However, make sure the location is added to your PATH environment variable for easy accessibility.

You can download it with the following command:

\`\`\`bash
curl -sSL https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash -s /usr/local/bin
\`\`\`

If you are using Arch Linux, you can install with pacman:

\`\`\`bash
pacman -S tgpt
\`\`\`


### FreeBSD 😈 

To install the [port](https://www.freshports.org/www/tgpt):
\`\`\`
cd /usr/ports/www/tgpt/ && make install clean
\`\`\`
To install the package, run one of these commands:
\`\`\`
pkg install www/tgpt
pkg install tgpt
\`\`\`

### Install with Go
You need to [add the Go install directory to your system's shell path](https://go.dev/doc/tutorial/compile-install). 

\`\`\`bash
go install github.com/aandrew-me/tgpt/v2@latest
\`\`\`

### Windows 🪟

-   **Scoop:** Package installation with [Scoop](https://scoop.sh/) can be done using the following command:

    \`\`\`bash
    scoop install https://raw.githubusercontent.com/aandrew-me/tgpt/main/tgpt.json
    \`\`\`

## Updating ⬆️
If you installed the program with the installation script, you may update it with
\`\`\`bash
tgpt -u
\`\`\`
**It may require admin privileges.**
### Proxy

Support:

1. environment variable

http_proxy or HTTP_PROXY with following available formats:

- Http Proxy [ \`http://ip:port\` ]
- Http Auth [ \`http://user:pass@ip:port\` ]
- Socks5 Proxy [ \`socks5://ip:port ]\`
- Socks5 Auth [ \`socks5://user:pass@ip:port\` ]

2. configuration file

file location in the following order:

- ./proxy.txt (in the same directory from where you are executing)
- ~/.config/tgpt/proxy.txt

Example:

\`\`\`bash
http://127.0.0.1:8080
\`\`\`

### From Release

You can download the executable for your operating system, rename it to \`tgpt\` (or any other desired name), and then execute it by typing \`./tgpt\` while in that directory. Alternatively, you can add it to your PATH environmental variable and then execute it by simply typing \`tgpt\`.


## Uninstalling
If you installed with the install script, you can execute the following command to remove the tgpt executable
\`\`\`
sudo rm $(which tgpt)
\`\`\`
Configuration file is usually located in \`~/.config/tgpt\` on GNU/Linux Systems and in \`"Library/Application Support/tgpt"\` on MacOS
`

try {
    writeFileSync("./README.md", readmeTxt);
    console.log("Updated README")
} catch (error) {
    console.log("Failed to update README", error)
}