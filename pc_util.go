package main

import (
	"log"

	pc "github.com/PeterXu/gopc"
)

type PeerConnection struct {
	TAG string

	stunName string
	tlsCrt   string
	tlsKey   string
	sdpOffer string

	connections map[string]*Connection // key: addr
	activeConn  *Connection
	dc          *pc.DcPeer
}

func NewPeerConnection(stunName, tlsCrt, tlsKey, offer string) *PeerConnection {
	var err error
	dc, err := pc.NewDcPeer(stunName, tlsCrt, tlsKey, "")
	if dc == nil {
		log.Println("fail to NewDcPeer for stunName:", stunName, err)
		return nil
	}
	pc := &PeerConnection{
		stunName:    stunName,
		tlsCrt:      tlsCrt,
		tlsKey:      tlsKey,
		sdpOffer:    offer,
		connections: make(map[string]*Connection),
		dc:          dc,
	}
	dc.ParseOfferSdp(offer)
	dc.Start(pc)
	return pc
}

func (pc *PeerConnection) addConnection(conn *Connection) {
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
		log.Println("no active conn, id=", pc.stunName)
		return nil
	}
	return pc.activeConn
}

func (pc *PeerConnection) onRecvDtlsData(data []byte) {
}

// callback of DcConnSink
func (pc *PeerConnection) OnDtlsStatus(err error, id string) {
}

// callback of DcConnSink
func (pc *PeerConnection) OnSctpStatus(err error, id string) {
}

// callback of DcConnSink
func (pc *PeerConnection) OnSctpData(data []byte, id string) {
}

// callback of DcConnSink
func (pc *PeerConnection) OnRtpRtcpData(data []byte, id string) {
}

// callback of DcConnSink
func (pc *PeerConnection) ToSendData(data []byte, id string) bool {
	return false
}
