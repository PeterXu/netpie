package main

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

func newSignalResponse(sequence string) *SignalResponse {
	return &SignalResponse{
		Sequence: sequence,
	}
}

type SignalResponse struct {
	Event    string // default not event
	Sequence string
	Result   []string
	Error    error

	conn *SignalConnection
}

/**
 * SignalMessage
 */
func newSignalMessage() *SignalMessage {
	return &SignalMessage{
		ctime: NowTimeMs(),
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
		Ctime: NowTimeMs(),
	}
}

type SignalService struct {
	Name        string
	Owner       string
	Started     bool
	Description string
	PwdMd5      string `json:"-"`
	Salt        string `json:"-"`
	Ctime       int64  `json:"-"`
}
