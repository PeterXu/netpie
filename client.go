package main

import (
	util "github.com/PeterXu/goutil"
)

// listen in local port(tcp/udp) to receive data, feeding to source.
func NewClient(sigaddr string) *Client {
	c := &Client{}
	c.Init(sigaddr)
	c.TAG = "client"
	return c
}

type Client struct {
	util.Logging
	signal *SignalEndpoint
}

func (c *Client) OnEvent(event SignalEvent) {
	c.Println("onEvent", event)
}

func (c *Client) Init(sigaddr string) {
	c.signal = NewSignalEndpoint(c)
	c.signal.Init(sigaddr, c)
}

func (c *Client) PreRunSignal(params []string) error {
	return nil
}

func (c *Client) PostRunSignal(params []string, err error) {
	if err != nil {
		c.Println("Run err:", err, params)
	}
}

func (c *Client) StartShell() {
	c.signal.StartShell("client")
}
