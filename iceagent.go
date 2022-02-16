package main

import (
	"context"

	util "github.com/PeterXu/goutil"
	ice "github.com/pion/ice/v2"
)

var (
	defaultStunUrls = []string{"stun.voipbuster.com", "stun.wirlab.net"}
)

type IceAgent struct {
	util.Logging
	*EvObject

	agent         *ice.Agent
	isControlling bool
	ch_send       chan []byte
	ch_recv       chan []byte
	ch_err        chan error
}

func NewIceAgent(controlling bool) *IceAgent {
	agent := &IceAgent{
		EvObject:      NewEvObject(),
		isControlling: controlling,
		ch_send:       make(chan []byte),
		ch_recv:       make(chan []byte),
		ch_err:        make(chan error),
	}
	agent.TAG = "ice"
	return agent
}

func (a *IceAgent) Init(urls []string) error {
	a.Println("init urls:", urls)

	if len(urls) == 0 {
		urls = append(urls, defaultStunUrls...)
	}

	var iceUrls []*ice.URL
	for _, item := range urls {
		if url, err := ice.ParseURL(item); err == nil {
			iceUrls = append(iceUrls, url)
		} else {
			a.Warnln("parse url error:", item, err)
		}
	}

	config := &ice.AgentConfig{
		Urls: iceUrls,
		NetworkTypes: []ice.NetworkType{
			ice.NetworkTypeUDP4,
			ice.NetworkTypeTCP4,
		},
		Lite:               true,
		InsecureSkipVerify: true,
	}

	if agent, err := ice.NewAgent(config); err != nil {
		a.Warnln("create agent error:", err)
		return err
	} else {
		a.agent = agent
	}

	// Event fired when new candidates gathered
	if err := a.agent.OnCandidate(func(c ice.Candidate) {
		if c != nil {
			a.AsyncFireEvent("ice-candidate", evData{"candidate": c.Marshal()})
		} else {
			a.Warnln("one candidate is nil")
		}
	}); err != nil {
		a.Warnln("listen candidate error:", err)
		return err
	}

	// When ICE Connection state has change
	if err := a.agent.OnConnectionStateChange(func(c ice.ConnectionState) {
		a.Println("Connection State has changed: ", c.String())
	}); err != nil {
		a.Warnln("listen connection error:", err)
		return (err)
	}

	// Event fired when selected candidate-pair changed.
	if err := a.agent.OnSelectedCandidatePairChange(func(c1, c2 ice.Candidate) {
		a.Println("Selected CandidatePair has changed: ", c1.String(), c2.String())
	}); err != nil {
		a.Warnln("listen candidate-pair error:", err)
		return (err)
	}

	// Get the local auth details and send to remote peer
	if localUfrag, localPwd, err := a.GetLocalUserCredentials(); err != nil {
		a.Warnln("get local auth error:", err)
		return (err)
	} else {
		if err := a.GatherCandidates(); err != nil {
			a.Warnln("gather candidates error:", err)
			return (err)
		}
		a.AsyncFireEvent("ice-auth", evData{"ufrag": localUfrag, "pwd": localPwd})
		return nil
	}
}

func (a *IceAgent) Uninit() {
	if a.agent != nil {
		a.Stop()
		a.agent.Close()
	}
}

// Start the ICE Agent. One side must be controlled, and the other must be controlling
func (a *IceAgent) Start(remoteUfrag, remotePwd string) error {
	var err error
	var conn *ice.Conn
	if a.isControlling {
		conn, err = a.Dial(context.TODO(), remoteUfrag, remotePwd)
	} else {
		conn, err = a.Accept(context.TODO(), remoteUfrag, remotePwd)
	}
	if err != nil {
		a.Warnln("agent start error:", err)
		return (err)
	}

	// Send messages in a loop to the remote peer
	go func() {
		defer func() {
			conn.Close()
		}()

		for {
			select {
			case data, ok := <-a.ch_send:
				if !ok {
					a.Warnln("write, chan error")
					return
				}
				if _, err = conn.Write(data); err != nil {
					a.Warnln("write, conn send error:", err)
					return
				}
			case err := <-a.ch_err:
				a.Println("write, recv error:", err)
				return
			}
		}
	}()

	go func() {
		// Receive messages in a loop from the remote peer
		buf := make([]byte, 1500)
		for {
			n, err := conn.Read(buf)
			if err != nil {
				a.Warnln("read, conn error:", err)
				a.ch_err <- nil
				return
			}
			a.ch_recv <- buf[0:n]
		}
	}()

	return nil
}

func (a *IceAgent) Stop() {
	a.ch_err <- nil
}

func (a *IceAgent) AddRemoteCandidate(candidate string) error {
	c, err := ice.UnmarshalCandidate(candidate)
	if err != nil {
		a.Println(err)
		return err
	}

	if err := a.agent.AddRemoteCandidate(c); err != nil {
		a.Println(err)
		return err
	}
	return nil
}

func (a *IceAgent) GetLocalCandidates() ([]ice.Candidate, error) {
	return a.agent.GetLocalCandidates()
}

func (a *IceAgent) GetLocalUserCredentials() (frag string, pwd string, err error) {
	frag, pwd, err = a.agent.GetLocalUserCredentials()
	return
}

func (a *IceAgent) GetRemoteUserCredentials() (frag string, pwd string, err error) {
	frag, pwd, err = a.agent.GetRemoteUserCredentials()
	return
}

func (a *IceAgent) GetSelectedCandidatePair() (*ice.CandidatePair, error) {
	return a.agent.GetSelectedCandidatePair()
}

func (a *IceAgent) SetRemoteCredentials(remoteUfrag, remotePwd string) error {
	return a.agent.SetRemoteCredentials(remoteUfrag, remotePwd)
}

func (a *IceAgent) RestartIce(ufrag, pwd string) error {
	return a.agent.Restart(ufrag, pwd)
}

func (a *IceAgent) GatherCandidates() error {
	return a.GatherCandidates()
}

func (a *IceAgent) GetCandidatePairsStats() []ice.CandidatePairStats {
	return a.GetCandidatePairsStats()
}

func (a *IceAgent) GetLocalCandidatesStats() []ice.CandidateStats {
	return a.agent.GetLocalCandidatesStats()
}

func (a *IceAgent) GetRemoteCandidatesStats() []ice.CandidateStats {
	return a.agent.GetRemoteCandidatesStats()
}

func (a *IceAgent) Dial(ctx context.Context, remoteUfrag, remotePwd string) (*ice.Conn, error) {
	return a.agent.Dial(ctx, remoteUfrag, remotePwd)
}

func (a *IceAgent) Accept(ctx context.Context, remoteUfrag, remotePwd string) (*ice.Conn, error) {
	return a.agent.Accept(ctx, remoteUfrag, remotePwd)
}
