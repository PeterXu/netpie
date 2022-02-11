package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	util "github.com/PeterXu/goutil"
	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

type fnClientAction = func(params []string) error

// listen in local port(tcp/udp) to receive data, feeding to source.
func NewClient(sigaddr string) *Client {
	c := &Client{
		actions: make(map[string]fnClientAction),
	}
	c.Init(sigaddr)
	c.TAG = "client"
	return c
}

type Client struct {
	util.Logging

	signal  *SignalClient
	actions map[string]fnClientAction
}

func (c *Client) OnEvent(event SignalEvent) {
}

func (c *Client) Init(sigaddr string) {
	c.signal = NewSignalClient(c)
	c.signal.sigaddr = sigaddr
}

func (c *Client) StartCli() {
	fmt.Println("Please use `exit` or `Ctrl-D` to exit this program.")
	defer fmt.Println("Bye!")
	defer HandleTTYOnExit()

	cc := newClientCompleter()
	p := prompt.New(
		c.Executor,
		cc.Complete,
		prompt.OptionTitle("client: interactive cmdline"),
		prompt.OptionPrefix(">>> "),
		prompt.OptionInputTextColor(prompt.Blue),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
	)
	p.Run()
}

func (c *Client) Executor(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	} else if s == "quit" || s == "exit" {
		fmt.Println("Bye!")
		os.Exit(0)
		return
	}

	// TODO
	var parts []string
	for _, item := range strings.Split(s, " ") {
		parts = append(parts, strings.Trim(item, "\""))
	}
	err := c.GoRun(parts[0], parts[1:])
	fmt.Println(":", s, parts, err)
	return
}

func (c *Client) GoRun(cmd string, params []string) error {
	if fn, ok := c.signal.actions[cmd]; ok {
		return fn(params)
	} else {
		return errors.New("invalid command:" + cmd)
	}
}

/**
 * client completer
 */
func newClientCompleter() *ClientCompleter {
	cc := &ClientCompleter{}
	cc.Init()
	return cc
}

type ClientCompleter struct {
	suggest []prompt.Suggest
}

func (cc *ClientCompleter) Init() {
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

func (cc *ClientCompleter) Complete(d prompt.Document) []prompt.Suggest {
	word := d.GetWordBeforeCursor()
	if len(word) > 0 {
		return prompt.FilterHasPrefix(cc.suggest, word, true)
	} else {
		return []prompt.Suggest{}
	}
}
