package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	util "github.com/PeterXu/goutil"
	"github.com/gorilla/websocket"
)

type fnSignalServerAction = func(req *SignalRequest, resp *SignalResponse) error

/**
 * Signal peer
 */
func newSignalPeer(id string, pwd_md5, salt string) *SignalPeer {
	return &SignalPeer{
		Id:         id,
		PwdMd5:     pwd_md5,
		Salt:       salt,
		InServices: make(map[string]bool),
	}
}

type SignalPeer struct {
	Id         string
	PwdMd5     string
	Salt       string
	InServices map[string]bool // id=>..
}

/**
 * Signal service
 */
func newSignalService() *SignalService {
	return &SignalService{
		Ctime: NowTimeMs(),
	}
}

type SignalService struct {
	Name        string
	Description string
	Owner       string
	PwdMd5      string `json:"-"`
	Salt        string `json:"-"`
	Ctime       int64  `json:"-"`
}

type SignalDatabase struct {
	Peers    map[string]*SignalPeer    // id=>..
	Services map[string]*SignalService // name=>..
}

/**
 * Signal server, manage all peers/services
 */
func NewSignalServer() *SignalServer {
	server := &SignalServer{
		db: &SignalDatabase{
			Peers:    make(map[string]*SignalPeer),
			Services: make(map[string]*SignalService),
		},

		ch_connect: make(chan *SignalConnection),
		ch_close:   make(chan *SignalConnection),
		ch_receive: make(chan *SignalMessage),

		connections: make(map[*SignalConnection]bool),
		onlines:     make(map[string]*SignalConnection),
		actions:     make(map[string]fnSignalServerAction),
	}

	server.TAG = "sigserver"
	server.actions["register"] = server.Register
	server.actions["login"] = server.Login
	server.actions["logout"] = server.Logout
	server.actions["services"] = server.Services
	server.actions["myservices"] = server.MyServices
	server.actions["join-service"] = server.JoinService
	server.actions["leave-service"] = server.LeaveService
	server.actions["show-service"] = server.ShowService
	return server
}

type SignalServer struct {
	util.Logging

	db *SignalDatabase

	ch_connect chan *SignalConnection
	ch_close   chan *SignalConnection
	ch_receive chan *SignalMessage

	connections map[*SignalConnection]bool
	onlines     map[string]*SignalConnection // uid => ..
	actions     map[string]fnSignalServerAction
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
			ss.connections[conn] = true
		case conn := <-ss.ch_close:
			ss.Printf("close one connection:%v\n", conn)
			if _, ok := ss.connections[conn]; ok {
				delete(ss.connections, conn)
			} else {
				if _, ok := ss.onlines[conn.id]; ok {
					delete(ss.onlines, conn.id)
				}
			}
			close(conn.send)
		case msg := <-ss.ch_receive:
			ss.Printf("receive request from connection:%v\n", msg.req.conn)
			ss.OnMessage(msg.req)
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

func (ss *SignalServer) CheckOnline(id string) error {
	if _, ok := ss.onlines[id]; ok {
		return nil
	} else {
		return errFnPeerNotLogin(id)
	}
}

func (ss *SignalServer) OnMessage(req *SignalRequest) {
	var err error
	resp := newSignalResponse(req.Sequence)
	if fn, ok := ss.actions[strings.ToLower(req.Action)]; ok {
		err = fn(req, resp)
	} else {
		err = errors.New("invalid action: " + req.Action)
	}
	if err != nil {
		ss.Printf("process message err: %v, conn: %v\n", err, req.conn)
	}

	resp.Error = err
	req.conn.send <- resp
}

func (ss *SignalServer) Register(req *SignalRequest, resp *SignalResponse) error {
	if len(req.FromId) < 10 {
		return errFnPeerInvalidId(req.FromId)
	}

	if len(req.PwdMd5) < 36 || len(req.Salt) < 4 {
		return errFnInvalidPwd(req.PwdMd5 + ":" + req.Salt)
	}

	if _, ok := ss.db.Peers[req.FromId]; ok {
		return errFnPeerExist(req.FromId)
	} else {
		new_pwd_md5 := MD5SumPwdSaltGenerate(req.PwdMd5, req.Salt)
		peer := newSignalPeer(req.FromId, new_pwd_md5, req.Salt)
		ss.db.Peers[req.FromId] = peer
		ss.SyncToStorage()
		return nil
	}
}

func (ss *SignalServer) Login(req *SignalRequest, resp *SignalResponse) error {
	conn := req.conn
	ss.Printf("client login with connection:%v\n", conn)

	if peer, ok := ss.db.Peers[req.FromId]; !ok {
		return errFnPeerNotFound(req.FromId)
	} else {
		if !MD5SumPwdSaltVerify(req.PwdMd5, peer.PwdMd5, peer.Salt) {
			return errFnWrongPwd(req.PwdMd5 + ":" + peer.Salt + " != " + peer.PwdMd5)
		}
		for id, _ := range peer.InServices {
			peer.InServices[id] = false
		}
	}

	conn.id = req.FromId
	if _, ok := ss.connections[conn]; ok {
		delete(ss.connections, conn)
	}
	ss.onlines[conn.id] = conn
	return nil
}

func (ss *SignalServer) Logout(req *SignalRequest, resp *SignalResponse) error {
	conn := req.conn
	ss.Printf("client offline with connection:%v\n", conn)
	if _, ok := ss.connections[conn]; !ok {
		if _, ok := ss.onlines[conn.id]; ok {
			delete(ss.onlines, conn.id)
			ss.connections[conn] = true
		}
	}
	return nil
}

// return all services
func (ss *SignalServer) Services(req *SignalRequest, resp *SignalResponse) error {
	if err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	for id, _ := range ss.db.Services {
		resp.Result = append(resp.Result, id)
	}
	return nil
}

// return joined services
func (ss *SignalServer) MyServices(req *SignalRequest, resp *SignalResponse) error {
	if err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	if peer, ok := ss.db.Peers[req.FromId]; !ok {
		return errFnPeerNotFound(req.FromId)
	} else {
		for id, ok := range peer.InServices {
			if ok {
				resp.Result = append(resp.Result, id)
			}
		}
		return nil
	}
}

func (ss *SignalServer) JoinService(req *SignalRequest, resp *SignalResponse) error {
	if err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	if peer, ok := ss.db.Peers[req.FromId]; !ok {
		return errFnPeerNotFound(req.FromId)
	} else {
		if item, ok := ss.db.Services[req.ServiceName]; !ok {
			return errFnServiceNotFound(req.ServiceName)
		} else {
			if !MD5SumPwdSaltVerify(req.ServicePwdMd5, item.PwdMd5, item.Salt) {
				return errFnWrongPwd(req.ServicePwdMd5 + ":" + item.Salt + " != " + item.PwdMd5)
			}
			peer.InServices[req.ServiceName] = true
		}
	}

	return nil
}

func (ss *SignalServer) LeaveService(req *SignalRequest, resp *SignalResponse) error {
	if err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	if peer, ok := ss.db.Peers[req.FromId]; ok {
		peer.InServices[req.ServiceName] = false
	}
	return nil
}

func (ss *SignalServer) ShowService(req *SignalRequest, resp *SignalResponse) error {
	if err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	if item, ok := ss.db.Services[req.ServiceName]; !ok {
		return errFnServiceNotFound(req.ServiceName)
	} else {
		resp.Result = append(resp.Result, JsonEncode(item))
		return nil
	}
}
