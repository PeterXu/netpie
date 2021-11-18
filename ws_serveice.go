package main

import (
	"fmt"
	"log"

	"golang.org/x/net/websocket"
)

type MessageService struct {
	url string
	ws  *websocket.Conn
}

func (m *MessageService) connect(url, origin string) {
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}
	m.ws = ws
}

func (m *MessageService) sendMessage(data []byte) bool {
	if _, err := m.ws.Write([]byte(data)); err != nil {
		log.Println(err)
		return false
	}
	return true
}

func startWsService(url string) {
	origin := "http://localhost/"
	ws, err := websocket.Dial(url, "", origin)
	if err != nil {
		log.Fatal(err)
	}

	if _, err := ws.Write([]byte("hello, world!\n")); err != nil {
		log.Fatal(err)
	}
	var msg = make([]byte, 512)
	var n int
	if n, err = ws.Read(msg); err != nil {
		log.Fatal(err)
	}
	fmt.Printf("Received: %s.\n", msg[:n])
}
