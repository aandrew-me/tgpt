<p align="center"><img src="tgpt.svg"></p>

# Terminal GPT (tgpt) 🚀

![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aandrew-me/tgpt)
![GitHub release (latest by date)](https://img.shields.io/github/v/release/aandrew-me/tgpt)
![AUR version](https://img.shields.io/aur/version/tgpt-bin?label=AUR%3A%20tgpt-bin)
![Chocolatey Version](https://img.shields.io/chocolatey/v/tgpt)

tgpt is a cross-platform cli (commandline) tool that lets you use ChatGPT 3.5 in Terminal **without API KEYS**. It communicates with the Backend of [Bai chatbot](https://chatbot.theb.ai). Its written in Go.

# Usage 💬

```
tgpt "What is the purpose of life?"
```

![demo](https://user-images.githubusercontent.com/66430340/233759296-c4cf8cf2-0cab-48aa-9e84-40765b823282.gif)

# Installation ⏬

## Download for GNU/Linux 🐧 or MacOS 🍎

The default download location is /usr/local/bin. But you can change it in the command and use your own location. However make sure its in PATH, if you want it to be easily accessible.

You can download with this command :-

```
curl -sSL https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash -s /usr/local/bin
```

If you are using Arch Linux you can install the [AUR package](https://aur.archlinux.org/packages/tgpt-bin) with `paru`:

```
paru -S tgpt-bin
```

Or with `yay`

```
yay -S tgpt-bin
```

## With Go

```
go install github.com/aandrew-me/tgpt@latest
```

## Windows 🪟

-   ### Chocolatey

    It can be installed from [Chocolatey](https://community.chocolatey.org/packages/tgpt) with

    ```
    choco install tgpt
    ```

-   ### Scoop
    Package can be installed with [scoop](https://scoop.sh/) with the following command -

    ```
    scoop install https://raw.githubusercontent.com/aandrew-me/tgpt/main/tgpt.json
    ```

## From Release

You can download an executable for your Operating System, then rename it to tgpt or whatever you want. Then you can execute it by typing `./tgpt` being in that directory. Or you can add it to the Environmental Variable **PATH** and then you can execute it by just typing `tgpt`.

### If you liked this project, give it a star! ⭐
