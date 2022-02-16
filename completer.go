package main

import (
	"github.com/c-bata/go-prompt"
)

/**
 * shell completer
 */
func NewShellCompleter() *ShellCompleter {
	cc := &ShellCompleter{}
	return cc
}

type ShellCompleter struct {
	suggest []prompt.Suggest
}

func (cc *ShellCompleter) Init(isServer bool) {
	if isServer {
		cc.InitServer()
	} else {
		cc.InitClient()
	}
}

func (cc *ShellCompleter) InitClient() {
	cc.suggest = []prompt.Suggest{
		{Text: "status", Description: "usage: status (show status to sigserver)"},
		{Text: "connect", Description: "usage: connect sigaddr (to sigserver)"},
		{Text: "disconnect", Description: "usage: disconnect (to sigserver)"},

		{Text: "register", Description: "usage: register id pwd"},
		{Text: "login", Description: "usage: login id pwd"},
		{Text: "logout", Description: "usage: logout"},

		{Text: "services", Description: "usage: services (list all services)"},
		{Text: "myservices", Description: "usage: myservices (list joined services)"},
		{Text: "show-service", Description: "usage: show-service serviceName (show service info)"},
		{Text: "join-service", Description: "usage: join-service serviceName pwd"},
		{Text: "leave-service", Description: "usage: leave-service serviceName"},
		{Text: "connect-service", Description: "usage: connect-service serviceName"},
		{Text: "disconnect-service", Description: "usage: disconnect-service serviceName"},
	}
}

func (cc *ShellCompleter) InitServer() {
	cc.suggest = []prompt.Suggest{
		{Text: "status", Description: "usage: status (show status to sigserver)"},
		{Text: "connect", Description: "usage: connect sigaddr (to sigserver)"},
		{Text: "disconnect", Description: "usage: disconnect (to sigserver)"},

		{Text: "register", Description: "usage: register id pwd"},
		{Text: "login", Description: "usage: login id pwd"},
		{Text: "logout", Description: "usage: logout"},

		{Text: "myservices", Description: "usage: myservices (list my services)"},
		{Text: "show-service", Description: "usage: show-service serviceName (show service info)"},
		{Text: "create-service", Description: "usage: create-service serviceName pwd description}"},
		{Text: "remove-service", Description: "usage: remove-service serviceName (only owner)"},
		{Text: "enable-service", Description: "usage: enable-service serviceName (only owner)"},
		{Text: "disable-service", Description: "usage: disable-service serviceName (only owner)"},
	}
}

func (cc *ShellCompleter) Complete(d prompt.Document) []prompt.Suggest {
	word := d.GetWordBeforeCursor()
	if len(word) > 0 {
		return prompt.FilterHasPrefix(cc.suggest, word, true)
	} else {
		return []prompt.Suggest{}
	}
}
