package main

import (
	"fmt"
	"log"
	"time"

	gn "github.com/panjf2000/gnet"
)

type udpService struct {
	*gn.EventServer
	tick time.Duration
}

func (s *udpService) OnInitComplete(srv gn.Server) (action gn.Action) {
	log.Printf("UDP service is listening on %s (multi-cores: %t, loops: %d)\n",
		srv.Addr.String(), srv.Multicore, srv.NumEventLoop)
	return
}

func (s *udpService) Tick() (delay time.Duration, action gn.Action) {
	log.Println("UDP service tick trigger.")
	delay = s.tick
	return
}

func (s *udpService) React(frame []byte, c gn.Conn) (out []byte, action gn.Action) {
	if frame != nil && len(frame) > 0 {
		data := make([]byte, len(frame))
		copy(data, frame)
		fireEvent("udp", evData{"conn": c, "data": data})
	} else {
		log.Println("no frame from", c.RemoteAddr())
	}
	return
}

func startUdpService(port int, intervalMs int) {
	multicore := false
	ticker := false
	interval := time.Duration(intervalMs) * time.Millisecond
	if interval > 0 {
		ticker = true
	}
	reuseport := true

	service := &udpService{tick: interval}
	addr := fmt.Sprintf("udp://:%d", port)
	log.Fatal(gn.Serve(service, addr,
		gn.WithMulticore(multicore),
		gn.WithTicker(ticker),
		gn.WithReusePort(reuseport)))
}
