//go:build !windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

func suspendProcess() {
	restoreTerminal()
	_ = syscall.Kill(os.Getpid(), syscall.SIGTSTP)
	rawModeOn := exec.Command("stty", "raw", "-echo")
	rawModeOn.Stdin = os.Stdin
	_ = rawModeOn.Run()
}
