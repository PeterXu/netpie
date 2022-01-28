package main

import (
	"log"
	"net/rpc"
)

/**
 * Signal client
 */

type SignalClient struct {
	id   string
	cb   SignalEventCallback
	conn *rpc.Client
}

func NewSignalClient(cb SignalEventCallback) *SignalClient {
	return &SignalClient{
		cb: cb,
	}
}

func (sc *SignalClient) Start(addr string) error {
	client, err := rpc.DialHTTP("tcp", addr)
	if err != nil {
		log.Fatal("dialing:", err)
	}
	sc.conn = client
	return nil
}

func (sc *SignalClient) Close() {
	if sc.conn != nil {
		sc.conn.Close()
		sc.conn = nil
	}
}

func (sc *SignalClient) Register(id, pwd string) error {
	// md5sum(pwd), and server will stored re-md5 with salt
	pwd_md5, salt := MD5SumPwdSaltGenerate(pwd)
	req := NewSignalRequest(sc.id)
	req.pwd_md5 = pwd_md5
	req.salt = salt

	_, err := sc.SendRequest(CurrentFunction(), req)
	return err
}

func (sc *SignalClient) Login(id, pwd string) error {
	// md5sum(pwd), and server will re-md5 with stored salt
	req := NewSignalRequest(sc.id)
	req.pwd_md5 = MD5SumPwdGenerate(pwd)

	_, err := sc.SendRequest(CurrentFunction(), req)
	return err
}

func (sc *SignalClient) Logout() error {
	req := NewSignalRequest(sc.id)

	_, err := sc.SendRequest(CurrentFunction(), req)
	return err
}

func (sc *SignalClient) SendRequest(method string, req SignalRequest) (*SignalResponse, error) {
	rpcMethod := "SignalProcess." + method
	resp := NewSignalResponse()
	err := sc.conn.Call(rpcMethod, req, resp)
	log.Println(method, " err:", err)
	return resp, err
}
