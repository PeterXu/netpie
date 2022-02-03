package main

import (
	"bytes"
	"log"
	"net"
	"strings"
	"time"

	util "github.com/PeterXu/goutil"
)

type Connection struct {
	TAG string

	addr     net.Addr
	stunName string
	ready    bool
	pc       *PeerConnection

	stunRequesting         int
	hadStunChecking        bool
	hadStunBindingResponse bool

	*TimeInfo
}

func NewConnection(addr net.Addr, stunName string) *Connection {
	return &Connection{
		TAG:      "[CONN]",
		addr:     addr,
		stunName: stunName,
		TimeInfo: NewTimeInfo(),
	}
}

func (c Connection) getStunName() string {
	return c.stunName
}

func (c Connection) getRemoteAddr() net.Addr {
	return c.addr
}

func (c Connection) isReady() bool {
	return c.ready
}

func (c *Connection) sendData(data []byte) bool {
	return false
}

func (c *Connection) onReceivedData(data []byte) {
	c.updateTime()

	if util.IsStunPacket(data) {
		var msg util.IceMessage
		if err := msg.Read(data); err != nil {
			log.Println(c.TAG, "invalid stun message", err)
			return
		}

		log.Println(c.TAG, "handle stun message")
		switch msg.Dtype {
		case util.STUN_BINDING_REQUEST:
			attr := msg.GetAttribute(util.STUN_ATTR_USERNAME)
			if attr == nil {
				log.Println(c.TAG, "no stun attr of username")
				return
			}

			stunName := string(attr.(*util.StunByteStringAttribute).Data)
			items := strings.Split(stunName, ":")
			if len(items) != 2 {
				log.Println(c.TAG, "invalid stun name:", stunName)
				return
			}
			c.onRecvStunBindingRequest(msg.TransId)
		case util.STUN_BINDING_RESPONSE:
			if c.hadStunBindingResponse {
				log.Println(c.TAG, "had stun binding response")
				return
			}
			log.Println(c.TAG, "recv stun binding response")
			c.hadStunBindingResponse = true
			c.ready = true
		case util.STUN_BINDING_ERROR_RESPONSE:
			log.Println(c.TAG, "error stun message")
		default:
			log.Println(c.TAG, "invalid stun type =", msg.Dtype)
		}
	} else {
		c.ready = true
		if c.pc != nil {
			c.pc.onRecvDtlsData(data)
		}
	}
}

func (c *Connection) onRecvStunBindingRequest(transId string) {
	if !c.isReady() {
		log.Println(c.TAG, "had left or not ready!")
		return
	}

	//log.Println(c.TAG, "send stun binding response")
	sendPwd := ""

	var buf bytes.Buffer
	if err := util.GenStunMessageResponse(&buf, sendPwd, transId, c.addr); err != nil {
		log.Println(c.TAG, "fail to gen stun response", err)
		return
	}

	//log.Println(c.TAG, "stun response len=", len(buf.Bytes()))
	c.sendData(buf.Bytes())
	c.checkStunBindingRequest()
}

func (c *Connection) sendStunBindingRequest() bool {
	if c.hadStunBindingResponse {
		return false
	}

	//log.Println(c.TAG, "send stun binding request")
	var sendUfrag string
	var recvUfrag, recvPwd string

	var buf bytes.Buffer
	if err := util.GenStunMessageRequest(&buf, sendUfrag, recvUfrag, recvPwd); err == nil {
		log.Println(c.TAG, "send stun binding request, len=", buf.Len())
		c.sendData(buf.Bytes())
	} else {
		log.Println(c.TAG, "fail to get stun request bufffer", err)
	}
	return true
}

func (c *Connection) checkStunBindingRequest() {
	if !c.sendStunBindingRequest() {
		return
	}

	if c.hadStunChecking {
		return
	}

	c.hadStunChecking = true
	go func() {
		c.stunRequesting = 500
		for {
			select {
			case <-time.After(time.Millisecond * time.Duration(c.stunRequesting)):
				if !c.sendStunBindingRequest() {
					log.Println(c.TAG, "quit stun request interval")
					c.hadStunChecking = false
					return
				}

				if delta := c.TimeInfo.sinceLastUpdate(); delta >= (15 * 1000) {
					log.Println(c.TAG, "no response from client and quit")
					return
				} else if delta > (5 * 1000) {
					log.Println(c.TAG, "adjust stun request interval")
					c.stunRequesting = delta / 2
				} else if delta < 500 {
					c.stunRequesting = 500
				}
			}
		}
	}()
}
