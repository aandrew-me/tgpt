<p align="center"><img src="tgpt.svg"></p>

# Terminal GPT (tgpt) 🚀

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt/releases/latest)
[![AUR version](https://img.shields.io/aur/version/tgpt-bin?label=AUR%3A%20tgpt-bin)](https://aur.archlinux.org/packages/tgpt-bin)
[![Chocolatey Version](https://img.shields.io/chocolatey/v/tgpt)](https://community.chocolatey.org/packages/tgpt)

tgpt is a cross-platform command-line interface (CLI) tool that allows you to use ChatGPT 3.5 in your Terminal without requiring API keys. It communicates with the backend of [Bai chatbot](https://chatbot.theb.ai) and is written in Go.

## Usage 💬

```bash
tgpt "What can you do?"
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
go install github.com/aandrew-me/tgpt@latest
```

### Windows 🪟

-   **Chocolatey:** You can install tgpt from [Chocolatey](https://community.chocolatey.org/packages/tgpt) using the following command:

    ```bash
    choco install tgpt
    ```

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

### From Release

You can download the executable for your operating system, rename it to `tgpt` (or any other desired name), and then execute it by typing `./tgpt` while in that directory. Alternatively, you can add it to your PATH environmental variable and then execute it by simply typing `tgpt`.

If you find this project useful, please give it a star! ⭐
