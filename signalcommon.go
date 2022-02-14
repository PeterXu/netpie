package main

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
	Sequence      string
	Action        string
	FromId        string
	PwdMd5        string
	Salt          string
	ServiceName   string
	ServicePwdMd5 string
	conn          *SignalConnection
}

func newSignalResponse(sequence string) *SignalResponse {
	return &SignalResponse{
		Sequence: sequence,
	}
}

type SignalResponse struct {
	Sequence string
	Result   []string
	Error    error
}

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
