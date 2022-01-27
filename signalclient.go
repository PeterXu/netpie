package main

/**
 * Signal client
 */

type SignalClient struct {
	cb   SignalEventCallback
	conn SignalConnection
	id   string
}

func NewSignalClient(cb SignalEventCallback) *SignalClient {
	return &SignalClient{cb: cb}
}

func (sc *SignalClient) Init() {
}

func (sc *SignalClient) Register(id, pwd string) {
	// md5sum(pwd), and server will stored re-md5 with salt
	pwd_md5_salt := MD5SumPwdSaltGenerate(pwd)
	msg := NewSignalMessage("register")
	msg.data = pwd_md5_salt
	sc.SendMessage(msg)
}

func (sc *SignalClient) Login(id, pwd string) {
	// md5sum(pwd), and server will re-md5 with stored salt
	pwd_md5 := MD5SumPwdGenerate(pwd)
	msg := NewSignalMessage("login")
	msg.data = pwd_md5
	sc.SendMessage(msg)
}

func (sc *SignalClient) Logout() {
	msg := NewSignalMessage("logout")
	sc.SendMessage(msg)
}

func (sc *SignalClient) SendMessage(msg *SignalMessage) {
	msg.fromId = sc.id
	sc.conn.SendData(msg.Serialize())
}

func (sc *SignalClient) OnReceviedMessage(msg *SignalMessage) {
}
