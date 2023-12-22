#!/bin/bash

echo "
1). Build for Linux
2). Build for Windows
3). Build for MacOS
4). Build for Linux, Windows and MacOS"

read -p "Please choose One [ 1-4 ]: " chs
if [[ $chs == "1" ]]; then
  # For GNU Linux
  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-amd64
  GOARCH=386 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-i386
  GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-arm64
elif [[ $chs == "2" ]]; then
  # For Windows
  GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-amd64.exe
  GOOS=windows GOARCH=386 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-i386.exe
  GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-arm64.exe
elif [[ $chs == "3" ]]; then
  # For MacOS
  GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-mac-amd64
  GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-mac-arm64
elif [[ $chs == "4" ]]; then
  # For GNU Linux
  GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-amd64
  GOARCH=386 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-i386
  GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-arm64

  # For Windows
  GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-amd64.exe
  GOOS=windows GOARCH=386 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-i386.exe
  GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-arm64.exe

  # For MacOS
  GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-mac-amd64
  GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-mac-arm64
else
  echo "Invalid Options !"
  exit 1
fi
