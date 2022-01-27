package main

import (
	"bytes"
	"encoding/gob"
	"log"
)

/**
 * Signal connection interface
 */
type SignalConnection interface {
	SendError(err string)
	SendSuccess(msg string)
	SendData(data []byte)
}

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
 * Signal message
 */
func NewSignalMessage(action string) *SignalMessage {
	return &SignalMessage{action: action}
}

type SignalMessage struct {
	action string
	fromId string
	toId   string
	data   string
}

func (m *SignalMessage) Parse(data []byte) bool {
	return true
}

func (m *SignalMessage) Serialize() []byte {
	buf := new(bytes.Buffer)
	enc := gob.NewEncoder(buf)
	if err := enc.Encode(m); err != nil {
		log.Println(err)
		return nil
	}
	return buf.Bytes()
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
	return &SignalGroup{}
}

type SignalGroup struct {
	id           string
	pwd_md5_salt string
	ctime        uint64
	utime        uint64
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

func (ss *SignalServer) Start() {
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

func (ss *SignalServer) OnReceivedMessage(conn SignalConnection, data []byte) {
	var msg SignalMessage
	switch msg.action {
	case "register":
	case "login":
	case "logout":
	default:
	}
}

func (ss *SignalServer) Register(conn SignalConnection, id string, pwd_md5_salt string) {
	if len(pwd_md5_salt) < 36 {
		conn.SendError("invalid password md5_salt!")
		return
	}

	if _, ok := ss.peers[id]; !ok {
		pwd_md5_salt2 := MD5SumPwdSaltReGenerate(pwd_md5_salt)
		if len(pwd_md5_salt2) > 0 {
			peer := NewSignalPeer(id, pwd_md5_salt2)
			ss.peers[id] = peer
			ss.SyncToStorage()
			conn.SendSuccess("register success")
		} else {
			conn.SendError("invalid pwd_md5_salt")
		}
	} else {
		conn.SendError("peer existed!")
	}
}

func (ss *SignalServer) IsPeerOnline(id string) bool {
	if peer, ok := ss.peers[id]; ok {
		return peer.online
	}
	return false
}

func (ss *SignalServer) Login(conn SignalConnection, id string, pwd_md5 string) {
	peer, ok := ss.peers[id]
	if !ok {
		conn.SendError("no this peer, id=" + id)
	} else {
		if !MD5SumPwdSaltVerify(pwd_md5, peer.pwd_md5_salt) {
			conn.SendError("wrong password for id=" + id)
		} else {
			peer.online = true
			conn.SendSuccess("login success")
		}
	}
}

func (ss *SignalServer) Logout(conn SignalConnection, id string) {
	if peer, ok := ss.peers[id]; ok {
		peer.online = false
		conn.SendSuccess("logout success")
	}
}
