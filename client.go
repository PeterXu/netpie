package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

// listen in local port(tcp/udp) to receive data, feeding to source.
func NewClient(sigaddr string) *Client {
	c := &Client{}
	c.Init(sigaddr)
	return c
}

type Client struct {
	signal *SignalClient
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
	signal := c.signal
	err := errors.New("invalid paramters")
	switch cmd {
	case "status":
		if len(params) == 0 {
			err = signal.Status()
		}
	case "connect":
		if len(params) == 1 {
			err = signal.Connect(params[0])
		}
	case "disconnect":
		if len(params) == 0 {
			err = signal.Disconnect()
		}
	case "register":
		if len(params) == 2 {
			err = signal.Register(params[0], params[1])
		}
	case "login":
		if len(params) == 2 {
			err = signal.Login(params[0], params[1])
		}
	case "logout":
		if len(params) == 0 {
			err = signal.Logout()
		}
	case "services":
		if len(params) == 0 {
			err = signal.Services()
		}
	case "join-service":
		if len(params) == 2 {
			err = signal.JoinService(params[0], params[1])
		}
	case "leave-service":
		if len(params) == 1 {
			err = signal.LeaveService(params[0])
		}
	case "show-service":
		if len(params) == 1 {
			err = signal.ShowService(params[0])
		}
	case "show-services":
		if len(params) == 0 {
			err = signal.ShowServices()
		}
	}
	return err
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
		{Text: "services", Description: "usage: services (list available services)"},
		{Text: "join-service", Description: "usage: join-service serviceName pwd"},
		{Text: "leave-service", Description: "usage: leave-service serviceName"},
		{Text: "show-service", Description: "usage: show-service serviceName (show service info)"},
		{Text: "show-services", Description: "usage: show-services (show joined services)"},
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
