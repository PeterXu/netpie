package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	gn "github.com/panjf2000/gnet"
)

type tcpService struct {
	*gn.EventServer
	tick          time.Duration
	clientSockets sync.Map
}

func (s *tcpService) OnInitComplete(srv gn.Server) (action gn.Action) {
	log.Printf("TCP service is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (s *tcpService) OnOpened(c gn.Conn) (out []byte, action gn.Action) {
	log.Printf("TCP socket with addr: %s has been opened...\n", c.RemoteAddr().String())
	s.clientSockets.Store(c.RemoteAddr().String(), c)
	return
}

func (s *tcpService) OnClosed(c gn.Conn, err error) (action gn.Action) {
	log.Printf("TCP socket with addr: %s is closing...\n", c.RemoteAddr().String())
	s.clientSockets.Delete(c.RemoteAddr().String())
	return
}

func (s *tcpService) Tick() (delay time.Duration, action gn.Action) {
	log.Println("TCP service tick trigger.")
	delay = s.tick
	return
}

func (s *tcpService) React(frame []byte, c gn.Conn) (out []byte, action gn.Action) {
	return
}

func startTcpService(port int, intervalMs int) {
	multicore := false
	ticker := false
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval > 0 {
		ticker = true
	}
	reuseport := true

	service := &tcpService{tick: interval}
	addr := fmt.Sprintf("tcp://:%d", port)
	log.Fatal(gn.Serve(service, addr,
		gn.WithMulticore(multicore),
		gn.WithTicker(ticker),
		gn.WithReusePort(reuseport)))
}
