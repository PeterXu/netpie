package main

import (
	"errors"
	"strings"
)

var (
	errNetworkHadConnected = errors.New("network had connected")
	errNetworkNotConnected = errors.New("network not connected")
	errRequestTimeout      = errors.New("request timeout")
	errWrongPassword       = errors.New("wrong password")
	errInvalidParameters   = errors.New("invalid paramters")
	errInvalidPassword     = errors.New("invalid password")
	errInvalidClientId     = errors.New("invalid client id")
	errClientNotLogin      = errors.New("client not login")
	errClientNotExist      = errors.New("client not exist")
	errClientExisted       = errors.New("client had existed")

	errFnInvalidParamters = func(args []string) error { return errors.New("invalid paramters:" + strings.Join(args, " ")) }
	errFnInvalidAction    = func(action string) error { return errors.New("invalid action:" + action) }

	errServiceNotExist       = errors.New("service not exist")
	errServiceExisted        = errors.New("service had existed")
	errServiceInvalidName    = errors.New("service invalid name")
	errServiceNotJoined      = errors.New("service not joined")
	errServiceShouldNotOwner = errors.New("service should not owner")
	errServiceRequireOwner   = errors.New("service require owner")

	errFnServiceInvalid = func(msg string) error { return errors.New("service invalid: " + msg) }
)

/**
 * Run result
 */
func NewResult(data string) *Result {
	return &Result{data: data}
}

type Result struct {
	data string
}
