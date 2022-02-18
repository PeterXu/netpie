package main

import (
	"errors"
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

func AnyToString(data interface{}) string {
	str, _ := data.(string)
	return str
}

func ParseCommandLine(line string) (parts []string, err error) {
	// special runes [ "'\].
	var lastCh rune
	var expectRightQuote bool

	var item string
	for idx, ch := range line {
		switch ch {
		case '\\': // only used in ".."
			if !expectRightQuote {
				err = fmt.Errorf("invalid char: <%s>", string(ch))
				return
			}
			if lastCh == '\\' {
				item += string(ch)
			}
		case '\'':
			if !expectRightQuote {
				err = fmt.Errorf("invalid char: <%s>", string(ch))
				return
			}
			item += string(ch)
		case ' ': // as split-char or used in ".."
			if expectRightQuote {
				item += string(ch)
			} else {
				if len(item) != 0 {
					parts = append(parts, item)
					item = ""
				}
			}
		case '"': // as field border or used in ".."
			if !expectRightQuote {
				expectRightQuote = true
			} else {
				if lastCh == '\\' {
					item += string(ch)
				} else {
					// support empty here
					expectRightQuote = false
					parts = append(parts, item)
					item = ""
				}
			}
		default:
			item += string(ch)
		}

		// when ended
		if len(line) == (idx + 1) {
			if expectRightQuote {
				err = errors.New("no right <\">")
				return
			}
			if len(item) > 0 {
				parts = append(parts, item)
				item = ""
			}
		}

		// save last char
		if lastCh == '\\' && ch == '\\' {
			lastCh = '?'
		} else {
			lastCh = ch
		}
	}
	return
}
