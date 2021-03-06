package main

import (
	"fmt"
	"net/http"
	"time"

	util "github.com/PeterXu/goutil"
	"github.com/gorilla/websocket"
)

/**
 * Signal connection
 *	a. incoming: recv data -> SignalRequest -> SignalMessage -> ...
 *  b. outgoing: send SignalResponse -> data -> ...,
 */

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

const (
	writeWait      = 3 * time.Second
	pongWait       = 60 * time.Second
	pingPeriod     = (pongWait * 9) / 10
	maxMessageSize = 1024
)

type SignalConnection struct {
	ss      *SignalServer
	conn    *websocket.Conn
	ch_send chan *SignalResponse
	id      string
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
		mt, data, err := c.conn.ReadMessage()
		c.ss.Println("conn, recv mt=", mt, len(data), err)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.ss.Printf("conn, close error: %v\n", err)
			}
			break
		}

		if mt == websocket.PongMessage {
			break
		}

		req := NewSignalRequest("")
		if err := util.GobDecode(data, req); err != nil {
			c.ss.Printf("conn, decode error: %v\n", err)
		} else {
			req.conn = c
			c.ss.ch_receive <- req
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
		case resp, ok := <-c.ch_send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The server closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if buf, err := util.GobEncode(resp); err != nil {
				c.ss.Printf("conn, encode err: %v\n", err)
			} else {
				if err := c.conn.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
					c.ss.Printf("conn, write err: %v\n", err)
				}
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				c.ss.Printf("conn, ping err: %v\n", err)
				return
			}
		}
	}
}

func serveWs(ss *SignalServer, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		ss.Println("conn, serverWs err", err)
		return
	}

	sconn := &SignalConnection{
		ss:      ss,
		conn:    conn,
		ch_send: make(chan *SignalResponse),
	}
	ss.ch_connect <- sconn

	go sconn.writePump()
	go sconn.readPump()
}
