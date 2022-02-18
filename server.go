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
	ep *Endpoint
}

func (s *Server) Init(sigaddr string) {
	s.ep = NewEndpoint(s, true)
	s.ep.Init(sigaddr)
}

func (s *Server) PreRunSignal(params []string) error {
	return nil
}

func (s *Server) PostRunSignal(params []string, err error) {
	if err == nil && len(params) > 0 {
		switch params[0] {
		case "enable-service":
			s.ep.CheckEnableLocalService("enable", params[1])
		case "disable-service":
			s.ep.CheckEnableLocalService("disable", params[1])
		}
	}
}

func (s *Server) StartShell() {
	s.ep.StartShell("server")
}
