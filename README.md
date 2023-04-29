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


## Legal Notice <a name="legal-notice"></a>

This repository uses third-party APIs and is *not* associated with or endorsed by the API providers. This project is intended **for educational purposes only**. This is just a little personal project.

Please note the following:

1. **Disclaimer**: The APIs, services, and trademarks mentioned in this repository belong to their respective owners. This project is *not* claiming any right over them.

2. **Responsibility**: The author of this repository is *not* responsible for any consequences arising from the use or misuse of this repository or the content provided by the third-party APIs and any damage or losses caused by users' actions.

3. **Educational Purposes Only**: This repository and its content are provided strictly for educational purposes. By using the information and code provided, users acknowledge that they are using the APIs and models at their own risk and agree to comply with any applicable laws and regulations.