package main

type EndpointCallback interface {
}

/**
 * 1. feeding data to endpoint and forwaded to remote endpoint.
 * 2. recving data from remote endpoint
 */

type Endpoint struct {
	cb EndpointCallback
}

func NewEndpoint(cb EndpointCallback) *Endpoint {
	return &Endpoint{
		cb: cb,
	}
}

func (ep *Endpoint) InputData(data []byte) {
}
