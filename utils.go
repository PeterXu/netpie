package main

import (
	"fmt"
	"os"
	"os/exec"

	"golang.org/x/term"
)

var termState *term.State

func SaveTermState() {
	oldState, err := term.GetState(int(os.Stdin.Fd()))
	if err != nil {
		return
	}
	termState = oldState
}

func RestoreTermState() {
	if termState != nil {
		term.Restore(int(os.Stdin.Fd()), termState)
	}
}

func ShellExecCmd(bin string, opts ...string) {
	name := "/bin/sh"
	args := append([]string{"-c", bin}, opts...)
	cmd := exec.Command(name, args...)

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Got error: %s\n", err.Error())
	}
}
