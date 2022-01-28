package main

import (
	"bytes"
	"encoding/gob"
	"errors"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

/**
 * Signal event
 */
func NewSignalEvent() *SignalEvent {
	return &SignalEvent{}
}

type SignalEvent struct {
}

type SignalEventCallback interface {
	OnEvent(event SignalEvent)
}

/**
 * Signal peer
 */
func NewSignalPeer(id string, pwd_md5_salt string) *SignalPeer {
	return &SignalPeer{
		id:           id,
		pwd_md5_salt: pwd_md5_salt,
		groups:       make(map[string]*SignalGroup),
	}
}

type SignalPeer struct {
	id           string
	pwd_md5_salt string
	groups       map[string]*SignalGroup
	online       bool
}

/**
 * Signal group
 */
func NewSignalGroup() *SignalGroup {
	return &SignalGroup{
		ctime: NowTimeMs(),
		utime: NowTimeMs(),
	}
}

type SignalGroup struct {
	id           string // group id
	pwd_md5_salt string // group pwd-md5
	ctime        int64  // group create time
	utime        int64  // group update time
}

/**
 * SignalRequest/SignalResponse
 */
func NewSignalRequest(id string) SignalRequest {
	return SignalRequest{
		from_id: id,
	}
}

type SignalRequest struct {
	from_id string
	pwd_md5 string
	salt    string
}

func NewSignalResponse() *SignalResponse {
	return &SignalResponse{}
}

type SignalResponse struct {
}

/**
 * Signal server, manage all peers/groups
 */
func NewSignalServer() *SignalServer {
	return &SignalServer{
		peers:  make(map[string]*SignalPeer),
		groups: make(map[string]*SignalGroup),
	}
}

type SignalServer struct {
	peers  map[string]*SignalPeer
	groups map[string]*SignalGroup
}

func (ss *SignalServer) Start(addr string) error {
	rpc.Register(NewSignalProcess(ss))
	rpc.HandleHTTP()

	l, e := net.Listen("tcp", addr)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	//rpc.Accept(l)

	go http.Serve(l, nil)
	return nil
}

func (ss *SignalServer) SyncToStorage() bool {
	var cached bytes.Buffer
	enc := gob.NewEncoder(&cached)
	if err := enc.Encode(ss); err != nil {
		log.Println(err)
		return false
	}
	return true
}

func (ss *SignalServer) SyncFromStorage() {
	var cached bytes.Buffer
	dec := gob.NewDecoder(&cached)
	if err := dec.Decode(&ss); err != nil {
		log.Println(err)
	}
}

func (ss *SignalServer) IsPeerOnline(id string) bool {
	if peer, ok := ss.peers[id]; ok {
		return peer.online
	}
	return false
}

/**
 * Signal Process
 */

func NewSignalProcess(server *SignalServer) *SignalProcess {
	return &SignalProcess{server: server}
}

type SignalProcess struct {
	server *SignalServer
}

func (sp *SignalProcess) Register(req SignalRequest, resp *SignalResponse) error {
	if len(req.from_id) < 10 {
		return errors.New("invalid id")
	}
	if len(req.pwd_md5) < 36 || len(req.salt) < 4 {
		return errors.New("invalid pwd_md5 and salt")
	}

	if _, ok := sp.server.peers[req.from_id]; !ok {
		pwd_md5_salt1 := req.pwd_md5 + ":" + req.salt
		pwd_md5_salt2 := MD5SumPwdSaltReGenerate(pwd_md5_salt1)
		if len(pwd_md5_salt2) > 0 {
			peer := NewSignalPeer(req.from_id, pwd_md5_salt2)
			sp.server.peers[req.from_id] = peer
			sp.server.SyncToStorage()
			return nil
		} else {
			return errors.New("invalid pwd_md5_salt")
		}
	} else {
		return errors.New("peer existed!")
	}
}

func (sp *SignalProcess) Login(req SignalRequest, resp *SignalResponse) error {
	peer, ok := sp.server.peers[req.from_id]
	if !ok {
		return errors.New("peer not exist, id=" + req.from_id)
	} else {
		if !MD5SumPwdSaltVerify(req.pwd_md5, peer.pwd_md5_salt) {
			return errors.New("wrong password for id=" + req.from_id)
		} else {
			peer.online = true
			return nil
		}
	}
}

func (sp *SignalProcess) Logout(req SignalRequest, resp *SignalResponse) error {
	if peer, ok := sp.server.peers[req.from_id]; ok {
		peer.online = false
	}
	return nil
}
