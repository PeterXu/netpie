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
	ep *Endpoint
}

func (c *Client) Init(sigaddr string) {
	c.ep = NewEndpoint(c, false)
	c.ep.Init(sigaddr)
}

func (c *Client) PreRunSignal(params []string) error {
	return nil
}

func (c *Client) PostRunSignal(params []string, err error) {
}

func (c *Client) StartShell() {
	c.ep.StartShell("client")
}
