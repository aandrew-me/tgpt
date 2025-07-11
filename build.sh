#!/bin/bash

echo "
1). Build for Linux
2). Build for Windows
3). Build for MacOS
4). Build for Linux, Windows and MacOS
0). Quit
"

read -rp "Please choose One [ 0-4 ]: " -n 1 chs
case $chs in
	1)
	# For GNU Linux
	CGO_ENABLED=0 GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-amd64
	CGO_ENABLED=0 GOARCH=386 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-i386
	CGO_ENABLED=0 GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-linux-arm64
	;;
	2)
	# For Windows
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-amd64.exe
	GOOS=windows GOARCH=386 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-i386.exe
	GOOS=windows GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-arm64.exe
	;;
	3)
	# For MacOS
	GOOS=darwin GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-mac-amd64
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags="-s -w" -o ./build/tgpt-mac-arm64
	;;
	4)
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
	;;
	0)
	exit 0
	;;
	*)
	echo "Invalid Options !"
	exit 1
	;;
esac
