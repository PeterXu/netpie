package main

import (
	ev "github.com/gookit/event"
)

type evEvent = ev.Event
type evData = ev.M

func listenEvent(name string, listener ev.Listener) {
	ev.Listen(name, listener, ev.Normal)
}

func fireEvent(name string, params ev.M) {
	ev.Fire(name, params)
}

func asyncFireEvent(name string, params ev.M) {
	go func() {
		fireEvent(name, params)
	}()
}
