package main

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	util "github.com/PeterXu/goutil"
	"github.com/gorilla/websocket"
)

/**
 * Signal client
 */
type fnSignalClientAction = func(action string, params []string) (*Result, error)

func NewSignalClient() *SignalClient {
	client := &SignalClient{
		EvObject: NewEvObject(),
		ch_send:  make(chan *SignalRequest, 3),
		ch_exit:  make(chan error),
		pending:  make(map[string]*SignalRequest),
		actions:  make(map[string]fnSignalClientAction),
	}

	client.TAG = "sigclient"
	client.actions[kActionStatus] = client.Status
	client.actions[kActionConnect] = client.Connect
	client.actions[kActionDisconnect] = client.Disconnect

	client.actions[kActionRegister] = client.Register
	client.actions[kActionLogin] = client.Login
	client.actions[kActionLogout] = client.Logout

	client.actions[kActionServices] = client.GoCheckService0
	client.actions[kActionMyServices] = client.GoCheckService0
	client.actions[kActionShowService] = client.GoCheckService1

	client.actions[kActionJoinService] = client.GoCheckService2
	client.actions[kActionLeaveService] = client.GoCheckService2
	client.actions[kActionConnectService] = client.GoCheckService2
	client.actions[kActionDisconnectService] = client.GoCheckService2

	client.actions[kActionCreateService] = client.GoCheckService3
	client.actions[kActionRemoveService] = client.GoCheckService2
	client.actions[kActionEnableService] = client.GoCheckService2
	client.actions[kActionDisableService] = client.GoCheckService2

	return client
}

type SignalClient struct {
	util.Logging
	*EvObject

	id      string
	ch_send chan *SignalRequest
	ch_exit chan error

	pending map[string]*SignalRequest
	actions map[string]fnSignalClientAction

	network NetworkStatus
	online  bool
	sigaddr string
}

func (sc *SignalClient) Start() {
	go func() {
		defer func() {
			sc.network = kNetworkDisconnected
		}()

		for {
			addr := sc.sigaddr
			sc.Println("client connecting to ", addr)
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

	sc.Println("run, connecting success")
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
				sc.Println("run, read fail:", err)
				sc.ch_exit <- err
				return
			} else {
				//sc.Println("run, read len:", len(data))
				resp := &SignalResponse{}
				if err := util.GobDecode(data, resp); err != nil {
					sc.Println("run, decode fail:", err)
				} else {
					sequence := resp.Sequence
					if len(resp.Event) == 0 {
						// this is request-response
						if item, ok := sc.pending[sequence]; ok {
							sc.Println("run, read response for seq:", sequence)
							item.ch_resp <- resp
							delete(sc.pending, sequence)
						} else {
							sc.Println("run, read not found seq:", sequence)
						}
					} else {
						// this is server event
						sc.Println("run, read event resp:", resp)
						sc.FireEvent(resp.Event, evData{"data": resp})
					}
				}
			}
		}
	}()

	// clean queue
	n := len(sc.ch_send)
	sc.Println("run, clean queue before write:", n)
	for i := 0; i < n; i++ {
		<-sc.ch_send
	}

	// write
	for {
		select {
		case req := <-sc.ch_send:
			if buf, err := util.GobEncode(req); err != nil {
				sc.Printf("run, encode fail: %v\n", err)
			} else {
				if err := c.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
					sc.Printf("run, write fail: %v\n", err)
					return err
				}
				if req.ch_resp != nil {
					sc.pending[req.Sequence] = req
				}
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
	if sc.network != kNetworkConnected {
		return errNetworkNotConnected
	}
	if sc.online && !expectOnline {
		return fmt.Errorf("user %s is logined", sc.id)
	} else if !sc.online && expectOnline {
		return fmt.Errorf("user %s not login", sc.id)
	} else {
		return nil
	}
}

/// network operations

func (sc *SignalClient) Status(action string, params []string) (*Result, error) {
	if len(params) != 0 {
		return nil, errFnInvalidParamters(params)
	}

	var result string
	switch sc.network {
	case kNetworkConnecting:
		result = "network connecting"
	case kNetworkConnected:
		if sc.CheckOnline(true) == nil {
			result = "network connected and onlined"
		} else {
			result = "network connected"
		}
	case kNetworkDisconnected:
		result = "network disconnected"
	default:
		result = "network unknown"
	}
	return NewResult(result), nil
}

func (sc *SignalClient) Connect(action string, params []string) (*Result, error) {
	if len(params) != 1 {
		return nil, errFnInvalidParamters(params)
	}
	sigaddr := params[0]

	if sc.network == kNetworkConnecting || sc.network == kNetworkConnected {
		result := "You need to disconnect at first!"
		return NewResult(result), errNetworkHadConnected
	} else {
		sc.sigaddr = sigaddr
		sc.Start()
		return nil, nil
	}
}

func (sc *SignalClient) Disconnect(action string, params []string) (*Result, error) {
	if len(params) != 0 {
		return nil, errFnInvalidParamters(params)
	}

	if sc.network == kNetworkConnecting || sc.network == kNetworkConnected {
		sc.Close()
		return nil, nil
	} else {
		return nil, errNetworkNotConnected
	}
}

/// user operations

func (sc *SignalClient) Register(action string, params []string) (*Result, error) {
	if len(params) != 2 {
		return nil, errFnInvalidParamters(params)
	}

	if err := sc.CheckOnline(false); err != nil {
		return nil, err
	}

	// md5sum(pwd), and server will stored re-md5 with salt
	req := NewSignalRequest(params[0])
	req.PwdMd5 = util.MD5SumGenerate([]string{params[1]})
	req.Salt = util.RandomString(4)
	if _, err := sc.SendRequest(action, req); err == nil {
		result := "Now you could login with them!"
		return NewResult(result), nil
	} else {
		return nil, err
	}
}

func (sc *SignalClient) Login(action string, params []string) (*Result, error) {
	if len(params) != 2 {
		return nil, errFnInvalidParamters(params)
	}

	if err := sc.CheckOnline(false); err != nil {
		return nil, err
	}

	// md5sum(pwd), and server will re-md5 with stored salt
	req := NewSignalRequest(params[0])
	req.PwdMd5 = util.MD5SumGenerate([]string{params[1]})
	if _, err := sc.SendRequest(action, req); err == nil {
		sc.id = req.FromId
		sc.online = true
		return nil, nil
	} else {
		return nil, err
	}
}

func (sc *SignalClient) Logout(action string, params []string) (*Result, error) {
	if len(params) != 0 {
		return nil, errFnInvalidParamters(params)
	}

	if err := sc.CheckOnline(true); err != nil {
		return nil, err
	}

	req := NewSignalRequest(sc.id)
	if _, err := sc.SendRequest(action, req); err == nil {
		sc.id = ""
		sc.online = false
		return nil, nil
	} else {
		return nil, err
	}
}

/// service operations

func (sc *SignalClient) GoCheckService0(action string, params []string) (*Result, error) {
	return sc.ControlService(action, params, 0)
}

func (sc *SignalClient) GoCheckService1(action string, params []string) (*Result, error) {
	return sc.ControlService(action, params, 1)
}

func (sc *SignalClient) GoCheckService2(action string, params []string) (*Result, error) {
	return sc.ControlService(action, params, 2)
}

func (sc *SignalClient) GoCheckService3(action string, params []string) (*Result, error) {
	return sc.ControlService(action, params, 3)
}

func (sc *SignalClient) ControlService(action string, params []string, count int) (*Result, error) {
	if len(params) != count {
		return nil, errFnInvalidParamters(params)
	}

	if err := sc.CheckOnline(true); err != nil {
		return nil, err
	}

	req := NewSignalRequest(sc.id)
	if count >= 1 {
		req.ServiceName = params[0]
	}
	if count >= 2 {
		req.ServicePwdMd5 = util.MD5SumGenerate([]string{params[1]})
		if action == kActionCreateService {
			req.ServiceSalt = util.RandomString(4)
		}
	}
	if count >= 3 {
		req.ServiceDesc = params[2]
	}

	if resp, err := sc.SendRequest(action, req); err == nil {
		result := strings.Join(resp.ResultL, "\n")
		return NewResult(result), nil
	} else {
		return nil, err
	}
}

/// ice message

func (sc *SignalClient) SendIceAuth(ufrag, pwd string, serviceName string) (*Result, error) {
	action := kActionEventIceAuth
	if err := sc.CheckOnline(true); err != nil {
		return nil, err
	}

	req := NewSignalRequest(sc.id)
	req.IceUfrag = ufrag
	req.IcePwd = pwd
	req.ServiceName = serviceName

	if resp, err := sc.SendRequest(action, req); err == nil {
		fmt.Println(resp)
		return nil, nil
	} else {
		return nil, err
	}
}

func (sc *SignalClient) SendIceCandidate(candidate string, serviceName string) (*Result, error) {
	action := kActionEventIceCandidate
	if err := sc.CheckOnline(true); err != nil {
		return nil, err
	}

	req := NewSignalRequest(sc.id)
	req.IceCandidate = candidate
	req.ServiceName = serviceName

	if resp, err := sc.SendRequest(action, req); err == nil {
		fmt.Println(resp)
		return nil, nil
	} else {
		return nil, err
	}
}

/// send request and wait response

func (sc *SignalClient) SendRequest(action string, req *SignalRequest) (*SignalResponse, error) {
	req.Action = action
	req.Sequence = util.RandomString(24)
	req.ch_resp = make(chan *SignalResponse, 2)

	sc.Println("send request:", req)

	ticker := time.NewTicker(3 * time.Second)
	defer func() {
		ticker.Stop()
	}()

	sc.ch_send <- req
	select {
	case resp, ok := <-req.ch_resp:
		if !ok {
			return nil, errors.New("closed by other")
		}
		if len(resp.Error) == 0 {
			return resp, nil
		} else {
			return nil, errors.New(resp.Error)
		}
	case <-ticker.C:
		return nil, errRequestTimeout
	}
}
