<img width="317" height="134" alt="tgpt logo" src="https://github.com/user-attachments/assets/ba582404-cd3a-4235-a451-c8a3cc5aef5a" />


# tgpt 🤖

[![GitHub go.mod Go version](https://img.shields.io/github/go-mod/go-version/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt)
[![GitHub release (latest by date)](https://img.shields.io/github/v/release/aandrew-me/tgpt)](https://github.com/aandrew-me/tgpt/releases/latest)
![Arch Linux package](https://img.shields.io/archlinux/v/extra/x86_64/tgpt)
![Chocolatey Version](https://img.shields.io/chocolatey/v/tgpt)
![Homebrew Formula Version](https://img.shields.io/homebrew/v/tgpt)


**tgpt** is a Cross-platform Command-Line Interface (CLI) tool that allows you to use AI in your Terminal.

<img src="https://github.com/user-attachments/assets/1b554b99-79ca-45b7-87ff-7713b7fd9437" alt="Demo" width="500" height="330">

### [Currently available providers](./md/providers.md)

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
#### Install with Homebrew
```bash
brew install tgpt
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
- #### Installation Script 
    Open Powershell, copy and paste the command and press Enter to install.
    ```
    irm https://raw.githubusercontent.com/aandrew-me/tgpt/refs/heads/main/install-win.ps1 | iex
    ```

    Uninstall with
    ```
    iex "& { $(irm https://raw.githubusercontent.com/aandrew-me/tgpt/refs/heads/main/install-win.ps1) } -Uninstall"
    ```


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

## [Usage](./md/usage.md) 

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
