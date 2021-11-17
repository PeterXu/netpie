package main

import (
	"log"

	"github.com/PeterXu/gopc"
)

type PeerConnection struct {
	stunName    string
	connections map[string]*Connection // key: addr
	activeConn  *Connection
	dc          *gopc.DcPeer
}

func NewPeerConnection(stunName, tslCrt, tlsKey, offer string) *PeerConnection {
	var err error
	dc, err := gopc.NewDcPeer(stunName, tlsCrt, tlsKey, "")
	if dc == nil {
		log.Errorln("fail to NewDcPeer for stunName:", stunName, err)
		return nil
	}
	dc.ParseOfferSdp(offer)
	dc.Start(u)
	return &PeerConnection{
		stunName:    stunName,
		connections: make(map[string]*Connection),
		dc:          dc,
	}
}

func (pc *PeerConnection) getActiveConn() *Connection {
	const TAG string = "[PC]"
	if pc.activeConn == nil {
		for k, v := range pc.connections {
			if v.isReady() {
				pc.activeConn = v
				log.Println("choose active conn, addr=", k, pc.stunName)
				break
			}
		}
	}
	if pc.activeConn == nil {
		log.Warnln("no active conn, id=", pc.stunName)
		return nil
	}
	return pc.activeConn
}
