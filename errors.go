package main

import (
	"errors"
	"strings"
)

var (
	errFnInvalidParamters = func(args []string) error { return errors.New("invalid paramters:" + strings.Join(args, " ")) }

	errFnInvalidPwd = func(pwd string) error { return errors.New("invalid pwd:" + pwd) }
	errFnWrongPwd   = func(pwd string) error { return errors.New("wrong pwd:" + pwd) }
	errFnInvalidId  = func(id string) error { return errors.New("invalid id: " + id) }

	errFnPeerExist    = func(id string) error { return errors.New("peer exist: " + id) }
	errFnPeerNotLogin = func(id string) error { return errors.New("peer not login: " + id) }
	errFnPeerNotFound = func(id string) error { return errors.New("peer not found: " + id) }

	errFnServiceInvalid     = func(msg string) error { return errors.New("service invalid: " + msg) }
	errFnServiceInvalidName = func(name string) error { return errors.New("service invalid name: " + name) }
	errFnServiceExist       = func(name string) error { return errors.New("service exist: " + name) }
	errFnServiceNotJoin     = func(name string) error { return errors.New("service not join: " + name) }
	errFnServiceNotExist    = func(name string) error { return errors.New("service not exist: " + name) }
	errFnServiceNotOwner    = func(id string) error { return errors.New("service not owner: " + id) }
	errFnServiceIsOwner     = func(id string) error { return errors.New("service is owner: " + id) }
)
