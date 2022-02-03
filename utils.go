package main

import (
	"bytes"
	"crypto/md5"
	"encoding/gob"
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

// return new "pwd_md5"
func MD5SumPwdSaltGenerate(pwd_md5, salt string) string {
	h := md5.New()
	io.WriteString(h, pwd_md5)
	io.WriteString(h, salt)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// verify pwd_md5 with salt
func MD5SumPwdSaltVerify(pwd_md5, stored_pwd_md5, salt string) bool {
	new_pwd_md5 := MD5SumPwdSaltGenerate(pwd_md5, salt)
	return (new_pwd_md5 == stored_pwd_md5)
}

func GoFunc() string {
	counter, _, _, success := runtime.Caller(1)
	if !success {
		//println("functionName: runtime.Caller: failed")
		//os.Exit(1)
		return "unknown"
	}

	fullname := runtime.FuncForPC(counter).Name()
	parts := strings.Split(fullname, ".")
	return strings.ToLower(parts[len(parts)-1])
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

func GobEncode(obj interface{}) (*bytes.Buffer, error) {
	cached := &bytes.Buffer{}
	enc := gob.NewEncoder(cached)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	} else {
		return cached, nil
	}
}

func GobDecode(data []byte, obj interface{}) error {
	cached := bytes.NewBuffer(data)
	dec := gob.NewDecoder(cached)
	return dec.Decode(obj)
}
