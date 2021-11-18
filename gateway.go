package main

import (
	"log"
	"net"
	"time"

	util "github.com/PeterXu/goutil"
)

var defaultGateway = NewGateway()

func NewGateway() *Gateway {
	gw := &Gateway{
		conferences: make(map[uint32]*Conference),
		stuns:       make(map[string]*StunInfo),
		connections: make(map[string]*Connection),
	}
	listenEvent("udp", gw)
	listenEvent("tcp", gw)
	go gw._loop()
	return gw
}

type Gateway struct {
	conferences map[uint32]*Conference
	stuns       map[string]*StunInfo
	pcs         map[string]*PeerConnection
	connections map[string]*Connection
}

func (g *Gateway) Handle(e evEvent) error {
	log.Println(e)
	switch e.Name() {
	case "udp":
		g.OnUdpPacket(e)
	case "tcp":
		g.OnTcpPacket(e)
	default:
	}
	return nil
}

func (g *Gateway) OnTcpPacket(e evEvent) {
}

func (g *Gateway) OnUdpPacket(e evEvent) {
	/*
		conn := e.Get("conn")
		data := e.Get("data")
		if sink := g.findConnection(conn.RemoteAddr()); sink != nil {
			sink.onReceivedData(data)
		} else {
			handleStunPacket(data, conn.RemoteAddr())
		}
	*/
}

func (g *Gateway) findConnection(addr net.Addr) *Connection {
	var key string = util.AddrToString(addr)
	if u, ok := g.connections[key]; ok {
		return u
	}
	return nil
}

func (g *Gateway) getPeerConnection(stunName string) *PeerConnection {
	if pc, ok := g.pcs[stunName]; ok {
		return pc
	}

	var tlsCrtPem string
	var tslKeyPem string
	var offer string
	return NewPeerConnection(stunName, tlsCrtPem, tslKeyPem, offer)
}

func (g *Gateway) _loop() {
	for {
		time.Sleep(time.Duration(10) * time.Second)
	}
}
