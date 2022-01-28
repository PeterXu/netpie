package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

// listen in local port(tcp/udp) to receive data, feeding to source.
func NewClient() *Client {
	return &Client{}
}

type Client struct {
	signal *SignalClient
}

func (c *Client) Start(sigaddr string) {
}

func (c *Client) CreateUser(id, pwd string) {
	c.signal.Register(id, pwd)
}

func (c *Client) ListService() {
}

func (c *Client) StartCli(sigaddr string) {
	fmt.Println("Please use `exit` or `Ctrl-D` to exit this program.")
	defer fmt.Println("Bye!")
	defer HandleTTYOnExit()

	cc := NewClientCompleter()
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
	return
}

func NewClientCompleter() *ClientCompleter {
	return &ClientCompleter{}
}

type ClientCompleter struct {
}

func (cc *ClientCompleter) Complete(d prompt.Document) []prompt.Suggest {
	s := []prompt.Suggest{
		{Text: "users", Description: "Store the username and age"},
		{Text: "articles", Description: "Store the article text posted by user"},
		{Text: "comments", Description: "Store the text commented to articles"},
	}
	return prompt.FilterHasPrefix(s, d.GetWordBeforeCursor(), true)
}
