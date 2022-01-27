package main

import (
	"crypto/md5"
	"fmt"
	"io"
	"math/rand"
	"strings"
	"time"
)

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

// return "pwd_md5:salt"
func MD5SumPwdSaltGenerate(pwd string) string {
	h := md5.New()
	io.WriteString(h, pwd)
	salt := RandomString(4)
	return fmt.Sprintf("%x:%s", h.Sum(nil), salt)
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
