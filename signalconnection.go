package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
)

/**
 * Signal connection
 *	a. incoming: recv data -> SignalRequest -> SignalMessage -> ...
 *  b. outgoing: send SignalResponse -> data -> ...,
 */

const (
	writeWait      = 10 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

type SignalConnection struct {
	ss   *SignalServer
	conn *websocket.Conn
	send chan *SignalResponse
	id   string
}

func (c SignalConnection) String() string {
	if c.conn != nil {
		return fmt.Sprintf("id=%s_raddr=%s", c.id, c.conn.RemoteAddr())
	} else {
		return fmt.Sprintf("id=%s_raddr=nil", c.id)
	}
}

func (c *SignalConnection) readPump() {
	defer func() {
		c.ss.ch_close <- c
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("conn, close error: %v\n", err)
			}
			break
		}

		req := newSignalRequest("")
		if err := GobDecode(data, req); err != nil {
			log.Printf("conn, decode error: %v\n", err)
		} else {
			msg := newSignalMessage()
			msg.req = req
			msg.conn = c
			c.ss.ch_receive <- msg
		}
	}
}

func (c *SignalConnection) writePump() {
	ticker := time.NewTicker(pingPeriod)

	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case resp, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The server closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if buf, err := GobEncode(resp); err != nil {
				log.Printf("conn, encode err: %v\n", err)
			} else {
				if err := c.conn.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
					log.Printf("conn, write err: %v\n", err)
				}
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				log.Printf("conn, ping err: %v\n", err)
				return
			}
		}
	}
}

func serveWs(ss *SignalServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("serverWs err", err)
		return
	}

	sconn := &SignalConnection{
		ss:   ss,
		conn: conn,
		send: make(chan *SignalResponse),
	}
	ss.ch_connect <- sconn

	go sconn.writePump()
	go sconn.readPump()
}