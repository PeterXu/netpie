package main

import (
	"errors"
	"net/http"
	"strings"
	"time"

	util "github.com/PeterXu/goutil"
)

/**
 * Signal database for storage
 */
type SignalDatabase struct {
	Peers    map[string]*SignalPeer    // id=>..
	Services map[string]*SignalService // name=>..
}

/**
 * Signal server, manage all peers/services
 */
type fnSignalServerAction = func(req *SignalRequest, resp *SignalResponse) error

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
	server.actions["show-service"] = server.ShowService

	server.actions["join-service"] = server.JoinService
	server.actions["leave-service"] = server.LeaveService
	server.actions["create-service"] = server.CreateService
	server.actions["remove-service"] = server.RemoveService
	server.actions["start-service"] = server.StartService
	server.actions["stop-service"] = server.StopService

	// ice-relative
	server.actions["ice-candidate"] = server.OnIceCandidate
	server.actions["ice-auth"] = server.OnIceAuth
	server.actions["ice-close"] = server.OnIceClose

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
				delete(ss.onlines, conn.id)
			}
			close(conn.ch_send)
		case msg := <-ss.ch_receive:
			ss.Printf("receive request from connection:%v\n", msg.req.conn)
			ss.OnMessage(msg.req)
		case <-tickChan.C:
			// clear timeout
		}
	}
}

func (ss *SignalServer) SyncToStorage() {
	if buf, err := util.GobEncode(ss.db); err != nil {
		ss.Printf("syncTo err: %v\n", err)
	} else {
		_ = buf
	}
}

func (ss *SignalServer) SyncFromStorage() {
	var cached []byte
	if err := util.GobDecode(cached, ss.db); err != nil {
		ss.Printf("syncFrom err:%v\n", err)
	}
}

func (ss *SignalServer) CheckOnline(id string) (*SignalPeer, error) {
	if _, ok := ss.onlines[id]; ok {
		if peer, ok := ss.db.Peers[id]; !ok {
			return nil, errFnPeerNotFound(id)
		} else {
			return peer, nil
		}
	} else {
		return nil, errFnPeerNotLogin(id)
	}
}

func (ss *SignalServer) CheckOnlineConn(id string) (*SignalConnection, error) {
	if conn, ok := ss.onlines[id]; ok {
		if _, ok := ss.db.Peers[id]; !ok {
			return nil, errFnPeerNotFound(id)
		} else {
			return conn, nil
		}
	} else {
		return nil, errFnPeerNotLogin(id)
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
	if resp.conn != nil {
		req.conn.ch_send <- newSignalResponse(req.Sequence)
		resp.conn.ch_send <- resp
	} else {
		req.conn.ch_send <- resp
	}
}

func (ss *SignalServer) Register(req *SignalRequest, resp *SignalResponse) error {
	if len(req.FromId) < 10 {
		return errFnInvalidId(req.FromId)
	}

	if len(req.PwdMd5) < 36 || len(req.Salt) < 4 {
		return errFnInvalidPwd(req.PwdMd5 + ":" + req.Salt)
	}

	if _, ok := ss.db.Peers[req.FromId]; ok {
		return errFnPeerExist(req.FromId)
	} else {
		new_pwd_md5 := util.MD5SumGenerate([]string{req.PwdMd5, req.Salt})
		peer := NewSignalPeer(req.FromId, new_pwd_md5, req.Salt)
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
		if !util.MD5SumVerify([]string{req.PwdMd5, peer.Salt}, peer.PwdMd5) {
			return errFnWrongPwd(req.PwdMd5 + ":" + peer.Salt + " != " + peer.PwdMd5)
		}

		// move from pending connections to onlines
		conn.id = req.FromId
		delete(ss.connections, conn)
		ss.onlines[conn.id] = conn
		return nil
	}
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
	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	for id := range ss.db.Services {
		resp.Result = append(resp.Result, id)
	}
	return nil
}

// return joined/created services
func (ss *SignalServer) MyServices(req *SignalRequest, resp *SignalResponse) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		for id, ok := range peer.InServices {
			if ok {
				resp.Result = append(resp.Result, id)
			}
		}
		return nil
	}
}

func (ss *SignalServer) ShowService(req *SignalRequest, resp *SignalResponse) error {
	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	if item, ok := ss.db.Services[req.ServiceName]; !ok {
		return errFnServiceNotExist(req.ServiceName)
	} else {
		resp.Result = append(resp.Result, util.JsonEncode(item))
		return nil
	}
}

func (ss *SignalServer) JoinService(req *SignalRequest, resp *SignalResponse) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if item, ok := ss.db.Services[req.ServiceName]; !ok {
			return errFnServiceNotExist(req.ServiceName)
		} else {
			if item.Owner == req.FromId {
				return errFnServiceInvalid("join not allowed")
			}

			if !util.MD5SumVerify([]string{req.ServicePwdMd5, item.Salt}, item.PwdMd5) {
				return errFnWrongPwd(req.ServicePwdMd5 + ":" + item.Salt + " != " + item.PwdMd5)
			}
			peer.InServices[req.ServiceName] = true
		}
	}

	return nil
}

func (ss *SignalServer) LeaveService(req *SignalRequest, resp *SignalResponse) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		peer.InServices[req.ServiceName] = false
	}
	return nil
}

func (ss *SignalServer) CreateService(req *SignalRequest, resp *SignalResponse) error {
	if len(req.ServiceName) == 0 {
		return errFnServiceInvalidName(req.ServiceName)
	}

	if len(req.ServicePwdMd5) < 36 || len(req.ServiceSalt) < 4 {
		return errFnInvalidPwd(req.ServicePwdMd5 + ":" + req.ServiceSalt)
	}

	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if _, ok := ss.db.Services[req.ServiceName]; ok {
			return errFnServiceExist(req.ServiceName)
		} else {
			service := NewSignalService(req.ServiceName, req.FromId)
			service.Description = req.ServiceDesc
			service.PwdMd5 = util.MD5SumGenerate([]string{req.ServicePwdMd5, req.ServiceSalt})
			service.Salt = req.ServiceSalt
			ss.db.Services[req.ServiceName] = service
			return nil
		}
	}
}

func (ss *SignalServer) RemoveService(req *SignalRequest, resp *SignalResponse) error {
	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if service, ok := ss.db.Services[req.ServiceName]; ok {
			if service.Owner != req.FromId {
				return errFnServiceNotOwner(req.FromId)
			}
			delete(ss.db.Services, req.ServiceName)

			// TODO: notify online peers?
			for _, item := range ss.db.Peers {
				delete(item.InServices, req.ServiceName)
			}
		}
		return nil
	}
}

func (ss *SignalServer) StartService(req *SignalRequest, resp *SignalResponse) error {
	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	if service, ok := ss.db.Services[req.ServiceName]; ok {
		if service.Owner != req.FromId {
			return errFnServiceNotOwner(req.FromId)
		}
		service.Started = true
		// TODO: notify
	}
	return nil
}

func (ss *SignalServer) StopService(req *SignalRequest, resp *SignalResponse) error {
	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	if service, ok := ss.db.Services[req.ServiceName]; ok {
		if service.Owner != req.FromId {
			return errFnServiceNotOwner(req.FromId)
		}
		service.Started = false
		// TODO: notify
	}
	return nil
}

func (ss *SignalServer) OnIceCandidate(req *SignalRequest, resp *SignalResponse) error {
	data := map[string]interface{}{
		"candidate": req.IceCandidate,
	}
	return ss.ForwardServiceData(req, resp, util.JsonEncode(data))
}

func (ss *SignalServer) OnIceAuth(req *SignalRequest, resp *SignalResponse) error {
	data := map[string]interface{}{
		"ice-ufrag": req.IceUfrag,
		"ice-pwd":   req.IcePwd,
	}
	return ss.ForwardServiceData(req, resp, util.JsonEncode(data))
}

func (ss *SignalServer) OnIceClose(req *SignalRequest, resp *SignalResponse) error {
	data := map[string]interface{}{}
	return ss.ForwardServiceData(req, resp, util.JsonEncode(data))
}

func (ss *SignalServer) ForwardServiceData(req *SignalRequest, resp *SignalResponse, jdata string) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if service, ok := ss.db.Services[req.ServiceName]; !ok {
			return errFnServiceNotExist(req.ServiceName)
		} else {
			var toId string
			if service.Owner == req.FromId {
				toId = req.ToId
			} else {
				if _, ok := peer.InServices[req.ServiceName]; !ok {
					return errFnServiceNotJoin(req.ServiceName)
				}
				toId = service.Owner
			}
			if conn, err := ss.CheckOnlineConn(toId); err != nil {
				return err
			} else {
				resp.Event = req.Action
				resp.Result = []string{jdata}
				resp.conn = conn
			}
		}
		return nil
	}
}
