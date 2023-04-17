#!/bin/bash

# For GNU Linux
go build -o ./build/tgpt-linux-x64
GOARCH=386 go build -o ./build/tgpt-linux-x86

# For Windows
GOOS=windows GOARCH=amd64 go build -o ./build/tgpt-x64.exe
GOOS=windows GOARCH=386 go build -o ./build/tgpt-x86.exe

# For MacOS
GOOS=darwin GOARCH=amd64 go build -o ./build/tgpt-mac-x64
GOOS=darwin GOARCH=arm64 go build -o ./build/tgpt-mac-arm64
