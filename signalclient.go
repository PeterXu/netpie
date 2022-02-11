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

type NetworkStatus int

const (
	kNetworkUnknown NetworkStatus = iota
	kNetworkConnecting
	kNetworkConnected
	kNetworkDisconnected
)

func NewSignalClient(cb SignalEventCallback) *SignalClient {
	client := &SignalClient{
		cb:      cb,
		send:    make(chan *SignalMessage),
		pending: make(map[string]*SignalMessage),
		exit:    make(chan error),
	}
	client.TAG = "sigclient"
	return client
}

type SignalClient struct {
	util.Logging

	id      string
	cb      SignalEventCallback
	send    chan *SignalMessage
	pending map[string]*SignalMessage
	exit    chan error

	network NetworkStatus
	online  bool
	sigaddr string
}

func (sc *SignalClient) Start() {
	go func() {
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
		sc.network = kNetworkDisconnected
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
				sc.exit <- err
				return
			} else {
				sc.Println("read, recv len", len(data))
				resp := &SignalResponse{}
				if err := GobDecode(data, resp); err != nil {
					sc.Println("read, decode fail", err)
				} else {
					sequence := resp.Sequence
					if item, ok := sc.pending[sequence]; ok {
						sc.Println("read, response for seq", sequence)
						item.ch_resp <- resp
						delete(sc.pending, sequence)
					} else {
						sc.Println("read, not found seq", sequence)
					}
				}
			}
		}
	}()

	// write
	for {
		select {
		case msg := <-sc.send:
			if buf, err := GobEncode(msg.req); err != nil {
				sc.Printf("write, encode fail: %v\n", err)
				break
			} else {
				if err := c.WriteMessage(websocket.BinaryMessage, buf.Bytes()); err != nil {
					sc.Printf("write, send fail: %v\n", err)
					return err
				}
				sc.pending[msg.req.Sequence] = msg
			}
		case err := <-sc.exit:
			return err
		case <-ticker.C:
			var seqs []string
			nowTime := NowTimeMs()
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

	return nil
}

func (sc *SignalClient) Close() {
	sc.Println("client close")
	sc.exit <- nil
}

func (sc *SignalClient) CheckOnline(expectOnline bool) error {
	if sc.online && !expectOnline {
		return errors.New(fmt.Sprintf("user %s is login", sc.id))
	} else if !sc.online && expectOnline {
		return errors.New(fmt.Sprintf("user %s not login", sc.id))
	} else {
		return nil
	}
}

func (sc *SignalClient) Status() error {
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

func (sc *SignalClient) Connect(sigaddr string) error {
	if sc.network == kNetworkConnecting || sc.network == kNetworkConnected {
		fmt.Println("connecting/connected: you need to disconnect at first")
	} else {
		sc.sigaddr = sigaddr
		sc.Start()
	}
	return nil
}

func (sc *SignalClient) Disconnect() error {
	sc.Close()
	return nil
}

func (sc *SignalClient) Register(id, pwd string) error {
	if err := sc.CheckOnline(false); err != nil {
		fmt.Println(err)
		return err
	}

	// md5sum(pwd), and server will stored re-md5 with salt
	req := newSignalRequest(sc.id)
	req.PwdMd5 = MD5SumPwdGenerate(pwd)
	req.Salt = RandomString(4)
	if _, err := sc.SendRequest(GoFunc(), req); err == nil {
		fmt.Println("register success and now you could login")
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) Login(id, pwd string) error {
	if err := sc.CheckOnline(false); err != nil {
		fmt.Println(err)
		return err
	}

	// md5sum(pwd), and server will re-md5 with stored salt
	req := newSignalRequest(id)
	req.PwdMd5 = MD5SumPwdGenerate(pwd)
	if _, err := sc.SendRequest(GoFunc(), req); err == nil {
		fmt.Println("login success")
		sc.id = id
		sc.online = true
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) Logout() error {
	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(err)
		return err
	}

	req := newSignalRequest(sc.id)
	if _, err := sc.SendRequest(GoFunc(), req); err == nil {
		fmt.Println("logout success")
		sc.online = false
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) Services() error {
	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(err)
		return err
	}

	req := newSignalRequest(sc.id)
	if resp, err := sc.SendRequest(GoFunc(), req); err == nil {
		fmt.Println(resp.Result)
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) MyServices() error {
	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(err)
		return err
	}

	req := newSignalRequest(sc.id)
	if _, err := sc.SendRequest(GoFunc(), req); err == nil {
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) JoinService(sid, pwd string) error {
	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(err)
		return err
	}

	req := newSignalRequest(sc.id)
	req.ServiceId = sid
	req.ServicePwdMd5 = MD5SumPwdGenerate(pwd)
	if _, err := sc.SendRequest(GoFunc(), req); err == nil {
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) LeaveService(sid string) error {
	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(err)
		return err
	}

	req := newSignalRequest(sc.id)
	if _, err := sc.SendRequest(GoFunc(), req); err == nil {
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) ShowService(sid string) error {
	if err := sc.CheckOnline(true); err != nil {
		fmt.Println(err)
		return err
	}

	req := newSignalRequest(sc.id)
	if _, err := sc.SendRequest(GoFunc(), req); err == nil {
		return nil
	} else {
		return err
	}
}

func (sc *SignalClient) SendRequest(action string, req *SignalRequest) (*SignalResponse, error) {
	req.Action = action
	req.Sequence = RandomString(24)

	ticker := time.NewTicker(3 * time.Second)
	ch_resp := make(chan *SignalResponse)
	defer func() {
		ticker.Stop()
		close(ch_resp)
	}()

	msg := newSignalMessage()
	msg.req = req
	msg.ch_resp = ch_resp
	sc.send <- msg

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
