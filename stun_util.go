package main

import (
	"log"
	"net"
	"strings"

	"github.com/PeterXu/gopc"
)

type StunInfo struct {
	cid   uint32
	uid   string
	offer string
	ctime int64
}

func handleStunPacket(data []byte, addr net.Addr) bool {
	if IsStunPacket(data) {
		handleStunBindingRequest(data, addr)
		return true
	} else {
		return false
	}
}

func handleStunBindingRequest(data []byte, addr net.Addr) {
	var msg gopc.IceMessage
	if !msg.Read(data) {
		log.Warnln("invalid stun message")
		return
	}

	log.Println("handle stun message")
	switch msg.Dtype {
	case gopc.STUN_BINDING_REQUEST:
		attr := msg.GetAttribute(gopc.STUN_ATTR_USERNAME)
		if attr == nil {
			log.Warnln("no stun attr of username")
			return
		}

		stunName := string(attr.(*gopc.StunByteStringAttribute).Data)
		items := strings.Split(stunName, ":")
		if len(items) != 2 {
			log.Warnln("invalid stun name:", stunName)
			return
		}

	default:
		log.Warnln("invalid stun type =", msg.Dtype)
	}
}
