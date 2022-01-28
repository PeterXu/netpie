package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"golang.org/x/term"
)

// NowMs return crrent UTC time(milliseconds) with 64bit
func NowTimeMs() int64 {
	return time.Now().UTC().UnixNano() / int64(time.Millisecond)
}

func RandomString(n int) string {
	rand.Seed(time.Now().UnixNano())

	var letter = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")
	b := make([]rune, n)
	for i := range b {
		b[i] = letter[rand.Intn(len(letter))]
	}
	return string(b)
}

// return "pwd_md5"
func MD5SumPwdGenerate(pwd string) string {
	h := md5.New()
	io.WriteString(h, pwd)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// return "pwd_md5, salt"
func MD5SumPwdSaltGenerate(pwd string) (string, string) {
	h := md5.New()
	io.WriteString(h, pwd)
	salt := RandomString(4)
	return fmt.Sprintf("%x", h.Sum(nil)), salt
}

// return "(pwd_md5:salt)_md5 : salt"
func MD5SumPwdSaltReGenerate(pwd_md5_salt string) string {
	parts := strings.Split(pwd_md5_salt, ":")
	if len(parts) != 2 {
		return ""
	}
	pwd_md5, salt := parts[0], parts[1]

	h := md5.New()
	io.WriteString(h, pwd_md5)
	io.WriteString(h, salt)
	return fmt.Sprintf("%x:%s", h.Sum(nil), salt)
}

func MD5SumPwdSaltVerify(pwd_md5, stored_pwd_md5_salt string) bool {
	parts := strings.Split(stored_pwd_md5_salt, ":")
	if len(parts) != 2 {
		return false
	}
	salt := parts[1]
	tmp_pwd_salt := MD5SumPwdSaltReGenerate(pwd_md5 + ":" + salt)
	return (tmp_pwd_salt == stored_pwd_md5_salt)
}

func CurrentFunction() string {
	counter, _, _, success := runtime.Caller(1)
	if !success {
		//println("functionName: runtime.Caller: failed")
		//os.Exit(1)
		return "unknown"
	}

	fullname := runtime.FuncForPC(counter).Name()
	parts := strings.Split(fullname, ".")
	return parts[len(parts)-1]
}

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

func HandleTTYOnExit() {
	rawModeOff := exec.Command("/bin/stty", "-raw", "echo")
	rawModeOff.Stdin = os.Stdin
	_ = rawModeOff.Run()
	rawModeOff.Wait()
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
