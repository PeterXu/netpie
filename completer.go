package main

import (
	"fmt"

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
		{Text: "help", Description: "usage: help"},

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
		{Text: "leave-service", Description: "usage: leave-service serviceName pwd"},
		{Text: "connect-service", Description: "usage: connect-service serviceName pwd"},
		{Text: "disconnect-service", Description: "usage: disconnect-service serviceName pwd"},
	}
}

func (cc *ShellCompleter) InitServer() {
	cc.suggest = []prompt.Suggest{
		{Text: "help", Description: "usage: help"},

		{Text: "status", Description: "usage: status (show status to sigserver)"},
		{Text: "connect", Description: "usage: connect sigaddr (to sigserver)"},
		{Text: "disconnect", Description: "usage: disconnect (to sigserver)"},

		{Text: "register", Description: "usage: register id pwd"},
		{Text: "login", Description: "usage: login id pwd"},
		{Text: "logout", Description: "usage: logout"},

		{Text: "services", Description: "usage: services (list all services)"},
		{Text: "myservices", Description: "usage: myservices (list my services)"},
		{Text: "show-service", Description: "usage: show-service serviceName (show service info)"},

		{Text: "create-service", Description: "usage: create-service serviceName pwd description}"},
		{Text: "remove-service", Description: "usage: remove-service serviceName pwd (only owner)"},
		{Text: "enable-service", Description: "usage: enable-service serviceName pwd (only owner)"},
		{Text: "disable-service", Description: "usage: disable-service serviceName pwd (only owner)"},
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

func (cc ShellCompleter) IsExist(cmd string) bool {
	for _, item := range cc.suggest {
		if item.Text == cmd {
			return true
		}
	}
	return false
}

func (cc ShellCompleter) PrintHelp() {
	fmt.Println("All avaiable commands: ")
	for _, item := range cc.suggest {
		fmt.Printf("  %s - %s\n", item.Text, item.Description)
	}
	fmt.Println("")
}
