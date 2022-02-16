package main

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	util "github.com/PeterXu/goutil"
	"github.com/gorilla/websocket"
)

/**
 * Signal client
 */
type fnSignalClientAction = func(action string, params []string) error

func NewSignalClient(cb SignalEventCallback) *SignalClient {
	client := &SignalClient{
		cb:      cb,
		ch_send: make(chan *SignalMessage),
		pending: make(map[string]*SignalMessage),
		ch_exit: make(chan error),
		actions: make(map[string]fnSignalClientAction),
	}

	client.TAG = "sigclient"
	client.actions["status"] = client.Status
	client.actions["connect"] = client.Connect
	client.actions["disconnect"] = client.Disconnect

	client.actions["register"] = client.Register
	client.actions["login"] = client.Login
	client.actions["logout"] = client.Logout

	client.actions["services"] = client.Services
	client.actions["myservices"] = client.MyServices
	client.actions["show-service"] = client.ShowService

	client.actions["join-service"] = client.JoinService
	client.actions["leave-service"] = client.LeaveService
	client.actions["create-service"] = client.CreateService
	client.actions["remove-service"] = client.RemoveService
	client.actions["start-service"] = client.StartService
	client.actions["stop-service"] = client.StopService

	return client
}

type SignalClient struct {
	util.Logging

	id      string
	cb      SignalEventCallback
	ch_send chan *SignalMessage
	pending map[string]*SignalMessage
	ch_exit chan error

	network NetworkStatus
	online  bool
	sigaddr string
	actions map[string]fnSignalClientAction
}

func (sc *SignalClient) Start() {
	go func() {
		defer func() {
			sc.network = kNetworkDisconnected
		}()

		for {
			addr := sc.sigaddr
			sc.Println("client connecting", addr)
			sc.network = kNetworkConnecting
			if err := sc.Run(addr); err != nil {
				sc.Println("client error and reconnect for err:", err)
				time.Sleep(3 * time.Second)
			} else {
				sc.Println("client exit")
				return
			}
		}
	}()
}

func (sc *SignalClient) Run(addr string) error {
	u := url.URL{Scheme: "ws", Host: addr, Path: "/ws"}
	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return err
	}
	sc.network = kNetworkConnected

	ticker := time.NewTicker(30 * time.Second)

	defer func() {
		sc.online = false
		c.Close()
		ticker.Stop()
	}()

	// read
	go func() {
		for {
			if _, data, err := c.ReadMessage(); err != nil {
				sc.Println("read, recv fail", err)
				sc.ch_exit <- err
				return
			} else {
				sc.Println("read, recv len", len(data))
				resp := &SignalResponse{}
				if err := util.GobDecode(data, resp); err != nil {
					sc.Println("read, decode fail", err)
				} else {
					sequence := resp.Sequence
					if len(resp.Event) == 0 {
						// this is request-response
						if item, ok := sc.pending[sequence]; ok {
							sc.Println("read, response for seq", sequence)
							item.ch_resp <- resp
							delete(sc.pending, sequence)
						} else {
							sc.Println("read, not found seq", sequence)
						}
					} else {
						// this is server event
						sc.Println("read, event data", resp)
					}
				}
			}
		}
	}()

	// write
	for {
		select {
		case msg := <-sc.ch_send:
			if buf, err := util.GobEncode(msg.req); err != nil {
				sc.Printf("write, encode fail: %v\n", err)
			} else {
				if err := c.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
					sc.Printf("write, send fail: %v\n", err)
					return err
				}
				sc.pending[msg.req.Sequence] = msg
			}
		case err := <-sc.ch_exit:
			return err
		case <-ticker.C:
			var seqs []string
			nowTime := util.NowMs()
			for k, v := range sc.pending {
				if nowTime > v.ctime+5*1000 {
					seqs = append(seqs, k)
				}
			}
			for _, s := range seqs {
				delete(sc.pending, s)
			}
		}
	}
}

func (sc *SignalClient) Close() {
	sc.Println("client close")
	sc.ch_exit <- nil
}

func (sc *SignalClient) CheckOnline(expectOnline bool) error {
	if sc.online && !expectOnline {
		return fmt.Errorf("user %s is login", sc.id)
	} else if !sc.online && expectOnline {
		return fmt.Errorf("user %s not login", sc.id)
	} else {
		return nil
	}
}

func (sc *SignalClient) SendRequest(action string, req *SignalRequest) (*SignalResponse, error) {
	req.Action = action
	req.Sequence = util.RandomString(24)

	ticker := time.NewTicker(3 * time.Second)
	ch_resp := make(chan *SignalResponse)
	defer func() {
		ticker.Stop()
		close(ch_resp)
	}()

	msg := newSignalMessage()
	msg.req = req
	msg.ch_resp = ch_resp
	sc.ch_send <- msg

	select {
	case resp := <-ch_resp:
		if resp.Error == nil {
			return resp, nil
		} else {
			return nil, resp.Error
		}
	case <-ticker.C:
		return nil, errors.New("request timeout")
	}
}

/// network operations

func (sc *SignalClient) Status(action string, params []string) error {
	if len(params) != 0 {
		return errFnInvalidParamters(params)
	}

	switch sc.network {
	case kNetworkConnecting:
		fmt.Println("connecting")
	case kNetworkConnected:
		fmt.Println("connected")
	case kNetworkDisconnected:
		fmt.Println("disconnected")
	default:
		fmt.Println("other status=", sc.network)
	}
	return nil
}

func (sc *SignalClient) Connect(action string, params []string) error {
	if len(params) != 1 {
		return errFnInvalidParamters(params)
	}
	sigaddr := params[1]

	if sc.network == kNetworkConnecting || sc.network == kNetworkConnected {
		fmt.Println("connecting/connected: you need to disconnect at first")
	} else {
		sc.sigaddr = sigaddr
		sc.Start()
	}
	return nil
}

func (sc *SignalClient) Disconnect(action string, params []string) error {
	if len(params) != 0 {
		return errFnInvalidParamters(params)
	}
	sc.Close()
	return nil
}

/// user operations

func (sc *SignalClient) Register(action string, params []string) error {
	if len(params) != 2 {
		return errFnInvalidParamters(params)
	}

	if err := sc.CheckOnline(false); err != nil {
		fmt.Println(err)
		return err
	}

	// md5sum(pwd), and server will stored re-md5 with salt
	req := newSignalRequest(params[0])
	req.PwdMd5 = util.MD5SumGenerate([]string{params[1]})
	req.Salt = util.RandomString(4)
	if _, err := sc.SendRequest(action, req); err == nil {
		fmt.Println("register success and now you could login")
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) Login(action string, params []string) error {
	if len(params) != 2 {
		return errFnInvalidParamters(params)
	}

	if err := sc.CheckOnline(false); err != nil {
		fmt.Println(err)
		return err
	}

	// md5sum(pwd), and server will re-md5 with stored salt
	req := newSignalRequest(params[0])
	req.PwdMd5 = util.MD5SumGenerate([]string{params[1]})
	if _, err := sc.SendRequest(action, req); err == nil {
		fmt.Println("login success")
		sc.id = req.FromId
		sc.online = true
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) Logout(action string, params []string) error {
	if len(params) != 0 {
		return errFnInvalidParamters(params)
	}

	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(err)
		return err
	}

	req := newSignalRequest(sc.id)
	if _, err := sc.SendRequest(action, req); err == nil {
		fmt.Println("logout success")
		sc.online = false
		return nil
	} else {
		return err
	}
}

/// service operations

func (sc *SignalClient) Services(action string, params []string) error {
	return sc.ControlService(action, params, 0)
}

func (sc *SignalClient) MyServices(action string, params []string) error {
	return sc.ControlService(action, params, 0)
}

func (sc *SignalClient) ShowService(action string, params []string) error {
	return sc.ControlService(action, params, 1)
}

func (sc *SignalClient) JoinService(action string, params []string) error {
	return sc.ControlService(action, params, 2)
}

func (sc *SignalClient) LeaveService(action string, params []string) error {
	return sc.ControlService(action, params, 1)
}

func (sc *SignalClient) CreateService(action string, params []string) error {
	return sc.ControlService(action, params, 3)
}

func (sc *SignalClient) RemoveService(action string, params []string) error {
	return sc.ControlService(action, params, 1)
}

func (sc *SignalClient) StartService(action string, params []string) error {
	return sc.ControlService(action, params, 1)
}

func (sc *SignalClient) StopService(action string, params []string) error {
	return sc.ControlService(action, params, 1)
}

func (sc *SignalClient) ControlService(action string, params []string, count int) error {
	if len(params) < count {
		return errFnInvalidParamters(params)
	}

	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(action, err)
		return err
	}

	req := newSignalRequest(sc.id)
	if count >= 1 {
		req.ServiceName = params[0]
	}
	if count >= 2 {
		req.ServicePwdMd5 = util.MD5SumGenerate([]string{params[1]})
		if action == "CreateService" {
			req.ServiceSalt = util.RandomString(4)
		}
	}
	if count >= 3 {
		req.ServiceDesc = params[2]
	}

	if resp, err := sc.SendRequest(action, req); err == nil {
		fmt.Println(resp)
		return nil
	} else {
		return err
	}
}

/// ice message

func (sc *SignalClient) SendIceCandidate(candidate string, serviceName string) error {
	action := "ice-candidate"
	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(action, err)
		return err
	}

	req := newSignalRequest(sc.id)
	req.IceCandidate = candidate
	req.ServiceName = serviceName
	if resp, err := sc.SendRequest(action, req); err == nil {
		fmt.Println(resp)
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) SendIceAuth(ufrag, pwd string, serviceName string) error {
	action := "ice-auth"
	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(action, err)
		return err
	}

	req := newSignalRequest(sc.id)
	req.IceUfrag = ufrag
	req.IcePwd = pwd
	req.ServiceName = serviceName
	if resp, err := sc.SendRequest(action, req); err == nil {
		fmt.Println(resp)
		return nil
	} else {
		return err
	}
}
