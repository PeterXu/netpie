package main

import (
	util "github.com/PeterXu/goutil"
)

const (
	kActionTesting = "testing"

	kActionStatus     = "status"
	kActionConnect    = "connect"
	kActionDisconnect = "disconnect"

	kActionRegister          = "register"
	kActionLogin             = "login"
	kActionLogout            = "logout"
	kActionServices          = "services"
	kActionMyServices        = "myservices"
	kActionShowService       = "show-service"
	kActionJoinService       = "join-service"
	kActionLeaveService      = "leave-service"
	kActionCreateService     = "create-service"
	kActionRemoveService     = "remove-service"
	kActionEnableService     = "enable-service"
	kActionDisableService    = "disable-service"
	kActionConnectService    = "connect-service"
	kActionDisconnectService = "disconnect-service"

	kActionEventIceOpen      = "ice-open"
	kActionEventIceOpenAck   = "ice-open-ack"
	kActionEventIceClose     = "ice-close"
	kActionEventIceCloseAck  = "ice-close-ack"
	kActionEventIceAuth      = "ice-auth"
	kActionEventIceCandidate = "ice-candidate"
)

/**
 * Network status
 */
type NetworkStatus int

const (
	kNetworkUnknown NetworkStatus = iota
	kNetworkConnecting
	kNetworkConnected
	kNetworkDisconnected
)

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

	FromId string
	PwdMd5 string
	Salt   string
	ToId   string

	ServiceName   string
	ServicePwdMd5 string
	ServiceDesc   string
	ServiceSalt   string

	IceCandidate string
	IceUfrag     string
	IcePwd       string

	conn *SignalConnection
}

func NewSignalResponse(sequence string) *SignalResponse {
	return &SignalResponse{
		Sequence: sequence,
		ResultM:  make(map[string]string),
	}
}

type SignalResponse struct {
	Event       string // default not event
	Sequence    string
	FromId      string
	ServiceName string

	ResultL []string
	ResultM map[string]string
	Error   string

	conn *SignalConnection
}

/**
 * SignalMessage
 */
func newSignalMessage() *SignalMessage {
	return &SignalMessage{
		ctime: util.NowMs(),
	}
}

type SignalMessage struct {
	req     *SignalRequest
	ch_resp chan *SignalResponse
	ctime   int64
}

/**
 * Signal peer
 */
func NewSignalPeer(id string, pwd_md5, salt string) *SignalPeer {
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
	InServices map[string]bool // name=>.., client join/leave
}

/**
 * Signal service
 */
func NewSignalService(name, id string) *SignalService {
	return &SignalService{
		Name:  name,
		Owner: id,
		Ctime: util.NowMs(),
	}
}

type SignalService struct {
	Name        string
	Owner       string
	Enabled     bool
	Description string
	PwdMd5      string `json:"-"`
	Salt        string `json:"-"`
	Ctime       int64  `json:"-"`
}
