#!/bin/bash

# For GNU Linux
go build -o ./build/tgpt-linux-amd64
GOARCH=386 go build -o ./build/tgpt-linux-i386

# For Windows
GOOS=windows GOARCH=amd64 go build -o ./build/tgpt-amd64.exe
GOOS=windows GOARCH=386 go build -o ./build/tgpt-i386.exe

# For MacOS
GOOS=darwin GOARCH=amd64 go build -o ./build/tgpt-mac-amd64
GOOS=darwin GOARCH=arm64 go build -o ./build/tgpt-mac-arm64
