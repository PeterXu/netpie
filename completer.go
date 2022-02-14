package main

import (
	"strings"

	"github.com/c-bata/go-prompt"
)

/**
 * shell completer
 */
func newShellCompleter() *ShellCompleter {
	cc := &ShellCompleter{}
	return cc
}

type ShellCompleter struct {
	suggest []prompt.Suggest
}

func (cc *ShellCompleter) Init(title string) {
	if strings.Contains(title, "client") {
		cc.InitClient()
	} else {
		cc.InitServer()
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
		{Text: "join-service", Description: "usage: join-service serviceName pwd"},
		{Text: "leave-service", Description: "usage: leave-service serviceName"},
		{Text: "show-service", Description: "usage: show-service serviceName (show service info)"},
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
		{Text: "create-service", Description: "usage: create-service serviceName pwd"},
		{Text: "remove-service", Description: "usage: remove-service serviceName pwd"},
		{Text: "start-service", Description: "usage: start-service serviceName pwd"},
		{Text: "stop-service", Description: "usage: stop-service serviceName"},
		{Text: "show-service", Description: "usage: show-service serviceName (show service info)"},
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
