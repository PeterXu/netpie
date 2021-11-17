package main

import "net"

type Connection struct {
	addr     net.Addr
	stunName string
}

func (c *Connection) onReceivedData(data []byte) {
	if handleStunPacket(data, c.addr) {
	} else {
		if dc := c.getDataChannel(c.stunName); dc != nil {
			dc.RecvDtlsPacket(data)
		}
	}
}

func (c *Connection) getDataChannel(name string) interface{} {
	return nil
}
