package main

import (
	"errors"
	"net/http"
	"time"

	util "github.com/PeterXu/goutil"
	"github.com/gorilla/websocket"
)

/**
 * Signal event
 */
func newSignalEvent() *SignalEvent {
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
func newSignalPeer(id string, pwd_md5, salt string) *SignalPeer {
	return &SignalPeer{
		id:      id,
		pwd_md5: pwd_md5,
		salt:    salt,
		groups:  make(map[string]*SignalGroup),
	}
}

type SignalPeer struct {
	id      string
	pwd_md5 string
	salt    string
	groups  map[string]*SignalGroup
	online  bool
}

/**
 * Signal group
 */
func newSignalGroup() *SignalGroup {
	return &SignalGroup{
		ctime: NowTimeMs(),
		utime: NowTimeMs(),
	}
}

type SignalGroup struct {
	id      string // group id
	pwd_md5 string // group pwd-md5
	salt    string // group salt
	ctime   int64  // group create time
	utime   int64  // group update time
}

/**
 * SignalRequest/SignalResponse
 */
func newSignalRequest(id string) *SignalRequest {
	return &SignalRequest{
		FromId: id,
	}
}

type SignalRequest struct {
	Sequence string
	Action   string
	FromId   string
	PwdMd5   string
	Salt     string
}

func newSignalResponse(sequence string) *SignalResponse {
	return &SignalResponse{
		Sequence: sequence,
	}
}

type SignalResponse struct {
	Sequence string
	Services []string
	Err      error
}

func newSignalMessage() *SignalMessage {
	return &SignalMessage{
		ctime: NowTimeMs(),
	}
}

type SignalMessage struct {
	req     *SignalRequest
	ch_resp chan *SignalResponse
	conn    *SignalConnection
	ctime   int64
}

/**
 * Signal server, manage all peers/groups
 */
func NewSignalServer() *SignalServer {
	server := &SignalServer{
		db: &SignalDatabase{
			peers:  make(map[string]*SignalPeer),
			groups: make(map[string]*SignalGroup),
		},
		ch_receive: make(chan *SignalMessage),
		ch_connect: make(chan *SignalConnection),
		ch_accept:  make(chan *SignalConnection),
		ch_close:   make(chan *SignalConnection),
		pending:    make(map[*SignalConnection]bool),
		conns:      make(map[string]*SignalConnection),
	}
	server.TAG = "sigserver"
	return server
}

type SignalDatabase struct {
	peers  map[string]*SignalPeer
	groups map[string]*SignalGroup
}

type SignalServer struct {
	util.Logging

	db *SignalDatabase

	// Inbound messages from the conns.
	ch_receive chan *SignalMessage

	ch_connect chan *SignalConnection
	ch_accept  chan *SignalConnection
	ch_close   chan *SignalConnection

	pending map[*SignalConnection]bool
	conns   map[string]*SignalConnection
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func (ss *SignalServer) Start(addr string) {
	go ss.Run()

	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(ss, w, r)
	})

	if err := http.ListenAndServe(addr, nil); err != nil {
		ss.Println("ListenAndServe err", err)
	}
}

func (ss *SignalServer) Run() {
	ss.Println("running begin")
	tickChan := time.NewTicker(time.Second * 30)
	for {
		select {
		case conn := <-ss.ch_connect:
			ss.Printf("add one connection: %v\n", conn)
			ss.pending[conn] = true
		case conn := <-ss.ch_accept:
			if len(conn.id) == 0 {
				ss.Printf("invalid client connection: %v\n", conn)
				break
			}
			ss.Printf("accept one connection:%v\n", conn)
			if _, ok := ss.pending[conn]; ok {
				delete(ss.pending, conn)
			}
			ss.conns[conn.id] = conn
		case conn := <-ss.ch_close:
			ss.Printf("close one connection:%v\n", conn)
			if _, ok := ss.pending[conn]; ok {
				delete(ss.pending, conn)
			} else {
				if _, ok := ss.conns[conn.id]; ok {
					delete(ss.conns, conn.id)
				}
			}
			close(conn.send)
		case msg := <-ss.ch_receive:
			ss.Printf("receive request from connection:%v\n", msg.conn)
			ss.OnMessage(msg.req, msg.conn)
		case <-tickChan.C:
			// clear timeout
		}
	}
}

func (ss *SignalServer) SyncToStorage() {
	if buf, err := GobEncode(ss.db); err != nil {
		ss.Printf("syncTo err: %v\n", err)
	} else {
		_ = buf
	}
}

func (ss *SignalServer) SyncFromStorage() {
	var cached []byte
	if err := GobDecode(cached, ss.db); err != nil {
		ss.Printf("syncFrom err:%v\n", err)
	}
}

func (ss *SignalServer) IsPeerOnline(id string) bool {
	if peer, ok := ss.db.peers[id]; ok {
		return peer.online
	}
	return false
}

func (ss *SignalServer) OnMessage(req *SignalRequest, conn *SignalConnection) {
	resp := newSignalResponse(req.Sequence)
	switch req.Action {
	case "register":
		resp.Err = ss.Register(req, resp)
	case "login":
		resp.Err = ss.Login(req, resp)
	case "logout":
		resp.Err = ss.Logout(req, resp)
	}
	if resp.Err != nil {
		ss.Printf("process message err: %v\n", resp.Err)
	}
	conn.send <- resp
}

func (ss *SignalServer) Register(req *SignalRequest, resp *SignalResponse) error {
	if len(req.FromId) < 10 {
		return errors.New("register invalid id=" + req.FromId)
	}

	if len(req.PwdMd5) < 36 || len(req.Salt) < 4 {
		return errors.New("register invalid pwd_md5 or salt")
	}

	if _, ok := ss.db.peers[req.FromId]; !ok {
		new_pwd_md5 := MD5SumPwdSaltGenerate(req.PwdMd5, req.Salt)
		peer := newSignalPeer(req.FromId, new_pwd_md5, req.Salt)
		ss.db.peers[req.FromId] = peer
		ss.SyncToStorage()
		return nil
	} else {
		return errors.New("register peer existed id=" + req.FromId)
	}
}

func (ss *SignalServer) Login(req *SignalRequest, resp *SignalResponse) error {
	peer, ok := ss.db.peers[req.FromId]
	if !ok {
		return errors.New("login peer not exist, id=" + req.FromId)
	} else {
		if !MD5SumPwdSaltVerify(req.PwdMd5, peer.pwd_md5, peer.salt) {
			return errors.New("login wrong password for id=" + req.FromId)
		} else {
			peer.online = true
			return nil
		}
	}
}

func (ss *SignalServer) Logout(req *SignalRequest, resp *SignalResponse) error {
	if peer, ok := ss.db.peers[req.FromId]; ok {
		peer.online = false
	}
	return nil
}
