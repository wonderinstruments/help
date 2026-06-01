package main

import (
	"os"
	"syscall"
)

func suppressStderr() {
	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		return
	}
	_ = syscall.Dup2(int(devNull.Fd()), 2)
	devNull.Close()
}
