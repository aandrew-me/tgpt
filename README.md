# Terminal GPT (tgpt)

tgpt is a cross-platform cli (commandline) tool that lets you use ChatGPT 3.5 in Terminal **without API KEYS**. It communicates with the Backend of [Bai chatbot](https://chatbot.theb.ai). Its written in Go.

# Usage
```
tgpt "What is the purpose of life?"
```
![demo](https://user-images.githubusercontent.com/66430340/233759296-c4cf8cf2-0cab-48aa-9e84-40765b823282.gif)

# Installation

## Download for Linux or Mac with this one line
```
curl -sSL https://raw.githubusercontent.com/aandrew-me/tgpt/main/install | bash
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

## Windows
The package has been submitted to `choco` and is waiting to be approved.
## From Release

You can download an executable for your Operating System, then rename it to tgpt or whatever you want. Then you can execute it by typing `./tgpt` being in that directory. Or you can add it to the Environmental Variable **PATH** and then you can execute it by just typing `tgpt`.
