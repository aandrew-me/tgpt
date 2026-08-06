//go:build windows

package main

func suspendProcess() {
	// Process suspension via SIGTSTP is not supported on Windows
}
