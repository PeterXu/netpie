package main

import (
	util "github.com/PeterXu/goutil"
)

/**
 * 1. connect to local service(e.g. ssh/http)
 * 2. geting data from Sink and forward to local service.
 */
func NewServer(sigaddr string) *Server {
	s := &Server{}
	s.Init(sigaddr)
	s.TAG = "server"
	return s
}

type Server struct {
	util.Logging
	signal *SignalEndpoint
}

func (s *Server) OnEvent(event SignalEvent) {
	s.Println("onEvent", event)
}

func (s *Server) Init(sigaddr string) {
	s.signal = NewSignalEndpoint(s)
	s.signal.Init(sigaddr, s)
}

func (s *Server) PreRunSignal(params []string) error {
	return nil
}

func (s *Server) PostRunSignal(params []string, err error) {
	if err != nil {
		s.Println("Run err:", err, params)
	}
}

func (s *Server) StartShell() {
	s.signal.StartShell("server")
}
