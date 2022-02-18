package main

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	util "github.com/PeterXu/goutil"
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

func ReadFile(fname string, maxSize int) ([]byte, error) {
	//content, err := ioutil.ReadFile("text.txt")
	file, err := os.Open(fname)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	if maxSize < 8*1024 {
		maxSize = 8 * 1024
	}
	buf := make([]byte, maxSize)

	total, err := file.Read(buf)
	if err != nil {
		if err != io.EOF {
			return nil, err
		}
	}

	return buf[:total], nil
}

func WriteFile(fname string, data []byte) error {
	file, err := os.Create(fname)
	if err != nil {
		return err
	}
	defer file.Close()

	total, err := file.Write(data)
	_ = total

	return err
}

func GenerateToken(id, pwdMd5 string) string {
	times := fmt.Sprintf("%d", util.NowMs())
	value := util.MD5SumGenerate([]string{id, pwdMd5, times})
	return fmt.Sprintf("%s_%s", value, times)
}

func VerifyToken(id, pwdMd5, token string) bool {
	parts := strings.Split(token, "_")
	if len(parts) == 2 {
		return util.MD5SumVerify([]string{id, pwdMd5, parts[1]}, parts[0])
	}
	return false
}

func CheckTokenTimeout(token string, timeout int) bool {
	parts := strings.Split(token, "_")
	if len(parts) == 2 {
		itime := util.Atoi64(parts[1])
		return itime+int64(timeout) >= util.NowMs()
	}
	return true
}

func VerifyTokenAndTime(id, pwdMd5, token string, timeout int) bool {
	if VerifyToken(id, pwdMd5, token) {
		return !CheckTokenTimeout(token, timeout)
	}
	return false
}
