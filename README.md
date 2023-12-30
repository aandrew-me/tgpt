<p align="center"><img src="tgpt.svg"></p>

# Terminal GPT (tgpt) 🚀

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt/releases/latest)
[![AUR version](https://img.shields.io/aur/version/tgpt-bin?label=AUR%3A%20tgpt-bin)](https://aur.archlinux.org/packages/tgpt-bin)

tgpt is a cross-platform command-line interface (CLI) tool that allows you to use AI chatbot in your Terminal without requiring API keys. 

Current model: llama-2-13b-chat

## Usage 💬

```bash
Usage: tgpt [Flag] [Prompt]

Flags:
-s, --shell                                        Generate and Execute shell commands. (Experimental) 
-c, --code                                         Generate Code. (Experimental)
-q, --quiet                                        Gives response back without loading animation
-w, --whole                                        Gives response back as a whole text
-img, --image                                      Generate images from text

Options:
-v, --version                                      Print version 
-h, --help                                         Print help message 
-i, --interactive                                  Start normal interactive mode 
-m, --multiline                                    Start multi-line interactive mode 
-cl, --changelog                                   See changelog of versions 
-u, --update                                       Update program 

Examples:
tgpt "What is internet?"
tgpt -m
tgpt -s "How to update my system?"
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

### Install with Go

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

### Proxy

Support:
- Http Proxy [ `http://ip:port` ]
- Http Auth [ `http://user:pass@ip:port` ]
- Socks5 Proxy [ `socks5://ip:port ]`
- Socks5 Auth [ `socks5://user:pass@ip:port` ]

If you want to use a proxy, create `proxy.txt` file in the same directory where the program is located and write your proxy configuration there.

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