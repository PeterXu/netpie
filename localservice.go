package main

import (
	"fmt"
	"time"

	gn "github.com/panjf2000/gnet"
)

func NewLocalService(name string, isServer bool) *LocalService {
	return &LocalService{
		name:     name,
		isServer: isServer,
	}
}

type LocalService struct {
	gn.EventServer

	name     string // serviceName
	isServer bool
	conn     gn.Conn
	client   *gn.Client

	agent *IceAgent
}

func (s *LocalService) Init(proto, addr string) error {
	if s.isServer {
		return s.InitServer(proto, addr)
	} else {
		return s.InitClient(proto, addr)
	}
}

func (s *LocalService) InitClient(proto, addr string) error {
	if cli, conn, err := startClient(proto, addr, s); err != nil {
		return err
	} else {
		s.conn = conn
		s.client = cli
		return nil
	}
}

func (s *LocalService) InitServer(proto, addr string) error {
	go startServer(proto, addr, s)
	return nil
}

func (s *LocalService) Uninit() {
	if s.conn != nil {
		s.conn.Close()
		s.conn = nil
	}
	if s.client != nil {
		s.client.Stop()
		s.client = nil
	}
	if s.agent != nil {
		s.agent.Uninit()
		s.agent = nil
	}
}

func (s *LocalService) InitIce(controlling bool, client *SignalClient) {
	s.agent = NewIceAgent(controlling)
	s.agent.Init([]string{})

	// listen ice-agent's events
	s.agent.ListenEvent("ice-auth", func(e evEvent) error {
		candidate := e.Get("candidate").(string)
		if len(candidate) > 0 {
			client.SendIceCandidate(candidate, s.name)
		}
		return nil
	})
	s.agent.ListenEvent("ice-candidate", func(e evEvent) error {
		ufrag := e.Get("ufrag").(string)
		pwd := e.Get("pwd").(string)
		if len(ufrag) > 0 && len(pwd) > 0 {
			client.SendIceAuth(ufrag, pwd, s.name)
		}
		return nil
	})
}

func (s *LocalService) OnIceAuth(ufrag, pwd string) error {
	return s.agent.Start(ufrag, pwd)
}

func (s *LocalService) OnIceCandidate(candidate string) error {
	return s.agent.AddRemoteCandidate(candidate)
}

func (s *LocalService) OnInitComplete(server gn.Server) (action gn.Action) {
	return
}

func (s *LocalService) OnOpened(conn gn.Conn) (out []byte, action gn.Action) {
	s.conn = conn
	return
}

func (s *LocalService) OnClosed(conn gn.Conn, err error) (action gn.Action) {
	s.conn = nil
	return
}

func (s *LocalService) React(frame []byte, conn gn.Conn) (out []byte, action gn.Action) {
	return
}

func (s *LocalService) Tick() (delay time.Duration, action gn.Action) {
	return
}

func startClient(proto, addr string, handler gn.EventHandler) (cli *gn.Client, conn gn.Conn, err error) {
	if cli, err = gn.NewClient(handler); err == nil {
		if err = cli.Start(); err == nil {
			conn, err = cli.Dial(proto, addr)
		}
	}
	return
}

func startServer(proto, addr string, handler gn.EventHandler) error {
	multicore := false
	useticker := true
	reuseport := true
	uri := fmt.Sprintf("%s://%s", proto, addr)
	return gn.Serve(handler, uri,
		gn.WithMulticore(multicore),
		gn.WithTicker(useticker),
		gn.WithReusePort(reuseport))
}
