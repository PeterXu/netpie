package main

import (
	"errors"
	"strings"
)

var (
	errFnInvalidParamters = func(args []string) error { return errors.New("invalid paramters:" + strings.Join(args, " ")) }

	errFnInvalidPwd = func(pwd string) error { return errors.New("invalid pwd:" + pwd) }
	errFnWrongPwd   = func(pwd string) error { return errors.New("wrong pwd:" + pwd) }

	errFnPeerInvalidId = func(id string) error { return errors.New("peer invalid id: " + id) }
	errFnPeerExist     = func(id string) error { return errors.New("peer exist: " + id) }
	errFnPeerNotLogin  = func(id string) error { return errors.New("peer not login: " + id) }
	errFnPeerNotFound  = func(id string) error { return errors.New("peer not found: " + id) }

	errFnServiceNotFound = func(id string) error { return errors.New("service not found: " + id) }
)
