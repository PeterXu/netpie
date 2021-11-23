package main

import (
	"context"
	"log"
	"net"
	"time"

	ice "github.com/pion/ice/v2"
	"github.com/pion/randutil"
)

type IceAgent struct {
	TAG           string
	agent         *ice.Agent
	isControlling bool
}

func NewIceAgent() *IceAgent {
	return &IceAgent{
		TAG: "ICE",
	}
}

func (ic *IceAgent) init(urls []string) error {
	log.Println("init agent", urls)
	var iceUrls []*ice.URL
	for i := range urls {
		if url, err := ice.ParseURL(urls[i]); err == nil {
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
		log.Println(err)
		return err
	}

	ic.agent = iceAgent

	// Event fired when new candidates gathered
	if err = iceAgent.OnCandidate(func(c ice.Candidate) {
		if c == nil {
			return
		}
		szval := c.Marshal()
		fireEvent("ice-candidate", evData{"data": szval}, ic.TAG)
	}); err != nil {
		log.Println(err)
		return err
	}

	// When ICE Connection state has change
	if err = iceAgent.OnConnectionStateChange(func(c ice.ConnectionState) {
		log.Println("ICE Connection State has changed: ", c.String())
	}); err != nil {
		return (err)
	}

	// Event fired when selected candidate-pair changed.
	if err = iceAgent.OnSelectedCandidatePairChange(func(c1, c2 ice.Candidate) {
		log.Println("ICE Selected CandidatePair has changed: ", c1.String(), c2.String())
	}); err != nil {
		return (err)
	}

	// Get the local auth details and send to remote peer
	localUfrag, localPwd, err := iceAgent.GetLocalUserCredentials()
	if err != nil {
		return (err)
	}

	fireEvent("ice-auth", evData{"ufrag": localUfrag, "pwd": localPwd}, ic.TAG)

	if err = iceAgent.GatherCandidates(); err != nil {
		return (err)
	}

	return nil
}

func (ic *IceAgent) start(remoteUfrag, remotePwd string) error {
	// Start the ICE Agent. One side must be controlled, and the other must be controlling
	iceAgent := ic.agent

	var err error
	var conn net.Conn
	if ic.isControlling {
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
			time.Sleep(time.Second * 3)

			val, err := randutil.GenerateCryptoRandomString(15,
				"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
			if err != nil {
				//panic(err)
				return
			}
			if _, err = conn.Write([]byte(val)); err != nil {
				//panic(err)
				return
			}

			log.Println("Sent: ", val)
		}
	}()

	// Receive messages in a loop from the remote peer
	buf := make([]byte, 1500)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			//panic(err)
			return err
		}

		log.Println("Received: ", string(buf[:n]))
	}

	return nil
}

func (ic *IceAgent) addRemoteCandidate(candidate string) error {
	c, err := ice.UnmarshalCandidate(candidate)
	if err != nil {
		log.Println(err)
		return err
	}

	if err := ic.agent.AddRemoteCandidate(c); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func (ic *IceAgent) setRemoteCredentials(remoteUfrag, remotePwd string) error {
	return ic.agent.SetRemoteCredentials(remoteUfrag, remotePwd)
}

func (ic *IceAgent) restartIce(ufrag, pwd string) error {
	return ic.agent.Restart(ufrag, pwd)
}

func (ic *IceAgent) getLocalCandidates() {
}

func (ic *IceAgent) getSelectedCandidatePair() {
}
