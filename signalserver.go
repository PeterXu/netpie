package main

import (
	"fmt"
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
	server.actions[kActionRegister] = server.Register
	server.actions[kActionLogin] = server.Login
	server.actions[kActionLogout] = server.Logout

	server.actions[kActionServices] = server.Services
	server.actions[kActionMyServices] = server.MyServices
	server.actions[kActionShowService] = server.ShowService

	server.actions[kActionJoinService] = server.JoinService
	server.actions[kActionLeaveService] = server.LeaveService
	server.actions[kActionCreateService] = server.CreateService
	server.actions[kActionRemoveService] = server.RemoveService
	server.actions[kActionEnableService] = server.CheckEnableService
	server.actions[kActionDisableService] = server.CheckEnableService
	server.actions[kActionConnectService] = server.CheckConnectService
	server.actions[kActionDisconnectService] = server.CheckConnectService

	// ice-relative
	server.actions[kActionEventIceOpen] = server.CheckOnIceStatus
	server.actions[kActionEventIceClose] = server.CheckOnIceStatus
	server.actions[kActionEventIceAuth] = server.OnIceAuth
	server.actions[kActionEventIceCandidate] = server.OnIceCandidate

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
			return nil, errClientNotExist
		} else {
			return peer, nil
		}
	} else {
		return nil, errClientNotLogin
	}
}

func (ss *SignalServer) CheckOnlineConn(id string) (*SignalConnection, error) {
	if conn, ok := ss.onlines[id]; ok {
		if _, ok := ss.db.Peers[id]; !ok {
			return nil, errClientNotExist
		} else {
			return conn, nil
		}
	} else {
		return nil, errClientNotLogin
	}
}

func (ss *SignalServer) OnMessage(req *SignalRequest) {
	ss.Printf("receive request: %s, seq: %s, conn: %v\n", req.Action, req.Sequence, req.conn)

	var err error
	resp := NewSignalResponse(req.Sequence)
	if fn, ok := ss.actions[strings.ToLower(req.Action)]; ok {
		err = fn(req, resp)
	} else {
		err = errFnInvalidAction(req.Action)
	}

	ss.Printf("complete request: %s, seq: %s, err: %v\n", req.Action, req.Sequence, err)
	if err != nil {
		resp.Error = fmt.Sprint(err)
	}

	if resp.conn != nil {
		req.conn.ch_send <- NewSignalResponse(req.Sequence)
		resp.conn.ch_send <- resp
	} else {
		req.conn.ch_send <- resp
	}
}

func (ss *SignalServer) Register(req *SignalRequest, resp *SignalResponse) error {
	if len(req.FromId) < 5 {
		ss.Warnf("client: %s, invalid id length\n", req.FromId)
		return errInvalidClientId
	}

	if len(req.PwdMd5) < 32 || len(req.Salt) < 4 {
		ss.Warnf("client: %s, invalid password: %s:%s\n", req.FromId, req.PwdMd5, req.Salt)
		return errInvalidPassword
	}

	if _, ok := ss.db.Peers[req.FromId]; ok {
		return errClientExisted
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
		return errClientNotExist
	} else {
		if !util.MD5SumVerify([]string{req.PwdMd5, peer.Salt}, peer.PwdMd5) {
			ss.Warnf("client: %s, wrong password: %s:%s != %s\n", req.FromId, req.PwdMd5, peer.Salt, peer.PwdMd5)
			return errWrongPassword
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
	} else {
		for name, srv := range ss.db.Services {
			if srv.Owner == req.FromId {
				resp.ResultL = append(resp.ResultL, fmt.Sprintf("%s - owned", name))
			} else {
				resp.ResultL = append(resp.ResultL, fmt.Sprintf("%s - owned %s", name, srv.Owner))
			}
		}
		return nil
	}
}

// return joined/created services
func (ss *SignalServer) MyServices(req *SignalRequest, resp *SignalResponse) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		for name, srv := range ss.db.Services {
			if srv.Owner == req.FromId {
				resp.ResultL = append(resp.ResultL, fmt.Sprintf("%s - owned", name))
			}
		}
		for name, ok := range peer.InServices {
			if ok {
				resp.ResultL = append(resp.ResultL, fmt.Sprintf("%s - joined", name))
			} else {
				resp.ResultL = append(resp.ResultL, fmt.Sprintf("%s - left", name))
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
		return errServiceNotExist
	} else {
		resp.ResultL = append(resp.ResultL, util.JsonEncode(item))
		return nil
	}
}

func (ss *SignalServer) CheckVerifyService(name string, pwdMd5 string) (*SignalService, error) {
	if service, ok := ss.db.Services[name]; !ok {
		return nil, errServiceNotExist
	} else {
		if !util.MD5SumVerify([]string{pwdMd5, service.Salt}, service.PwdMd5) {
			ss.Warnf("service:%s, wrong password: %s:%s != %s\n", name, pwdMd5, service.Salt, service.PwdMd5)
			return nil, errWrongPassword
		}
		return service, nil
	}
}

func (ss *SignalServer) JoinService(req *SignalRequest, resp *SignalResponse) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if service, err := ss.CheckVerifyService(req.ServiceName, req.ServicePwdMd5); err != nil {
			return err
		} else {
			if service.Owner == req.FromId {
				return errFnServiceInvalid("owner need not join")
			} else {
				peer.InServices[req.ServiceName] = true
			}
		}
	}

	return nil
}

func (ss *SignalServer) LeaveService(req *SignalRequest, resp *SignalResponse) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if _, err := ss.CheckVerifyService(req.ServiceName, req.ServicePwdMd5); err != nil {
			return err
		} else {
			if _, ok := peer.InServices[req.ServiceName]; ok {
				peer.InServices[req.ServiceName] = false
			}
			// TODO
		}
	}
	return nil
}

func (ss *SignalServer) CreateService(req *SignalRequest, resp *SignalResponse) error {
	if len(req.ServiceName) == 0 {
		return errServiceInvalidName
	}

	if len(req.ServicePwdMd5) < 32 || len(req.ServiceSalt) < 4 {
		ss.Warnf("service: %s, invalid password: %s:%s\n", req.ServiceName, req.ServicePwdMd5, req.ServiceSalt)
		return errInvalidPassword
	}

	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if _, ok := ss.db.Services[req.ServiceName]; ok {
			return errServiceExisted
		} else {
			service := NewSignalService(req.ServiceName, req.FromId)
			service.Description = req.ServiceDesc
			service.PwdMd5 = util.MD5SumGenerate([]string{req.ServicePwdMd5, req.ServiceSalt})
			service.Salt = req.ServiceSalt
			ss.db.Services[req.ServiceName] = service
			// TODO
			return nil
		}
	}
}

func (ss *SignalServer) RemoveService(req *SignalRequest, resp *SignalResponse) error {
	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if service, err := ss.CheckVerifyService(req.ServiceName, req.ServicePwdMd5); err != nil {
			return err
		} else {
			if service.Owner != req.FromId {
				return errServiceRequireOwner
			}
			delete(ss.db.Services, req.ServiceName)
			// TODO: notify
			for _, item := range ss.db.Peers {
				delete(item.InServices, req.ServiceName)
			}
			return nil
		}
	}
}

func (ss *SignalServer) CheckEnableService(req *SignalRequest, resp *SignalResponse) error {
	if _, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	}

	if service, err := ss.CheckVerifyService(req.ServiceName, req.ServicePwdMd5); err != nil {
		return err
	} else {
		if service.Owner != req.FromId {
			return errServiceRequireOwner
		}
		switch req.Action {
		case kActionEnableService:
			service.Enabled = true
		case kActionDisableService:
			service.Enabled = false
		}
		// TODO: notify
		return nil
	}
}

func (ss *SignalServer) CheckConnectService(req *SignalRequest, resp *SignalResponse) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if isIn, ok := peer.InServices[req.ServiceName]; !ok || !isIn {
			return errServiceNotJoined
		}
		if service, err := ss.CheckVerifyService(req.ServiceName, req.ServicePwdMd5); err != nil {
			return err
		} else {
			if service.Owner == req.FromId {
				return errServiceShouldNotOwner
			}
		}

		switch req.Action {
		case kActionConnectService:
			req.Action = kActionEventIceOpen
		case kActionDisconnectService:
			req.Action = kActionEventIceClose
		default:
			return errFnInvalidParamters([]string{req.Action})
		}
		return ss.CheckOnIceStatus(req, resp)
	}
}

func (ss *SignalServer) CheckOnIceStatus(req *SignalRequest, resp *SignalResponse) error {
	return ss.ForwardServiceData(req, resp)
}

func (ss *SignalServer) OnIceCandidate(req *SignalRequest, resp *SignalResponse) error {
	resp.ResultM["ice-candidate"] = req.IceCandidate
	return ss.ForwardServiceData(req, resp)
}

func (ss *SignalServer) OnIceAuth(req *SignalRequest, resp *SignalResponse) error {
	resp.ResultM["ice-ufrag"] = req.IceUfrag
	resp.ResultM["ice-pwd"] = req.IcePwd
	return ss.ForwardServiceData(req, resp)
}

func (ss *SignalServer) ForwardServiceData(req *SignalRequest, resp *SignalResponse) error {
	if peer, err := ss.CheckOnline(req.FromId); err != nil {
		return err
	} else {
		if service, ok := ss.db.Services[req.ServiceName]; !ok {
			return errServiceNotExist
		} else {
			var toId string
			if service.Owner == req.FromId {
				toId = req.ToId
			} else {
				if _, ok := peer.InServices[req.ServiceName]; !ok {
					return errServiceNotJoined
				}
				toId = service.Owner
			}
			if conn, err := ss.CheckOnlineConn(toId); err != nil {
				return err
			} else {
				resp.Event = req.Action
				resp.conn = conn
			}
		}
		return nil
	}
}
