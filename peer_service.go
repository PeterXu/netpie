package main

import (
	"log"
)

type PeerService struct {
}

func NewPeerService() *PeerService {
	c := &PeerService{}
	listenEvent("ice-candidate", c, "ICE")
	listenEvent("ice-auth", c, "ICE")
	return c
}

func (p *PeerService) Handle(e evEvent) error {
	log.Println(e)
	switch e.Name() {
	case "ice-candidate":
	case "ice-auth":
	}
	return nil
}

func (p *PeerService) init() {
}
