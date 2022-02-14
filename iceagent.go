package main

import (
	"context"
	"net"

	util "github.com/PeterXu/goutil"
	ice "github.com/pion/ice/v2"
)

var (
	defaultStunUrls = []string{"stun.voipbuster.com", "stun.wirlab.net"}
)

type IceAgent struct {
	util.Logging
	evObj         *EvObject
	agent         *ice.Agent
	isControlling bool
	ch_send       chan []byte
	ch_recv       chan []byte
	ch_err        chan error
}

func NewIceAgent(controlling bool) *IceAgent {
	agent := &IceAgent{
		evObj:         newEvObject(),
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

	var iceUrls []*ice.URL
	for _, item := range urls {
		if url, err := ice.ParseURL(item); err == nil {
			iceUrls = append(iceUrls, url)
		}
	}

	iceConfig := &ice.AgentConfig{
		Urls: iceUrls,
		NetworkTypes: []ice.NetworkType{
			ice.NetworkTypeUDP4,
			ice.NetworkTypeTCP4,
		},
		Lite:               true,
		InsecureSkipVerify: true,
	}

	iceAgent, err := ice.NewAgent(iceConfig)
	if err != nil {
		a.Println(err)
		return err
	}
	a.agent = iceAgent

	// Event fired when new candidates gathered
	if err = iceAgent.OnCandidate(func(c ice.Candidate) {
		if c != nil {
			szval := c.Marshal()
			a.evObj.fireEvent("ice-candidate", evData{"data": szval})
		}
	}); err != nil {
		a.Println(err)
		return err
	}

	// When ICE Connection state has change
	if err = iceAgent.OnConnectionStateChange(func(c ice.ConnectionState) {
		a.Println("Connection State has changed: ", c.String())
	}); err != nil {
		return (err)
	}

	// Event fired when selected candidate-pair changed.
	if err = iceAgent.OnSelectedCandidatePairChange(func(c1, c2 ice.Candidate) {
		a.Println("Selected CandidatePair has changed: ", c1.String(), c2.String())
	}); err != nil {
		return (err)
	}

	// Get the local auth details and send to remote peer
	localUfrag, localPwd, err := iceAgent.GetLocalUserCredentials()
	if err != nil {
		return (err)
	}

	a.evObj.fireEvent("ice-auth", evData{"ufrag": localUfrag, "pwd": localPwd})

	if err = iceAgent.GatherCandidates(); err != nil {
		return (err)
	}

	return nil
}

func (a *IceAgent) Start(remoteUfrag, remotePwd string) error {
	// Start the ICE Agent. One side must be controlled, and the other must be controlling
	iceAgent := a.agent

	var err error
	var conn net.Conn
	if a.isControlling {
		conn, err = iceAgent.Dial(context.TODO(), remoteUfrag, remotePwd)
	} else {
		conn, err = iceAgent.Accept(context.TODO(), remoteUfrag, remotePwd)
	}
	if err != nil {
		return (err)
	}

	// Send messages in a loop to the remote peer
	go func() {
		for {
			select {
			case data, ok := <-a.ch_send:
				if !ok {
					return
				}
				if _, err = conn.Write(data); err != nil {
					return
				}
			case err := <-a.ch_err:
				a.Println(err)
				return
			}
		}
	}()

	// Receive messages in a loop from the remote peer
	buf := make([]byte, 1500)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			a.ch_err <- nil
			return err
		}
		a.ch_recv <- buf[0:n]
	}
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

func (a *IceAgent) SetRemoteCredentials(remoteUfrag, remotePwd string) error {
	return a.agent.SetRemoteCredentials(remoteUfrag, remotePwd)
}

func (a *IceAgent) RestartIce(ufrag, pwd string) error {
	return a.agent.Restart(ufrag, pwd)
}

func (a *IceAgent) GetLocalCandidates() {
}

func (a *IceAgent) GetSelectedCandidatePair() {
}
