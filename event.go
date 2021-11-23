package main

import (
	ev "github.com/gookit/event"
)

type evEvent = ev.Event
type evData = ev.M

var gEvManagers map[string]*ev.Manager

func init() {
	gEvManagers = make(map[string]*ev.Manager)
}

func getEvManager(key string, createIfNo bool) *ev.Manager {
	if len(key) == 0 {
		return ev.DefaultEM
	} else {
		val, ok := gEvManagers[key]
		if !ok && createIfNo {
			val = ev.NewManager(key)
			gEvManagers[key] = val
		}
		return val
	}
}

func listenEvent(name string, listener ev.Listener, key string) {
	if obj := getEvManager(key, true); obj != nil {
		obj.Listen(name, listener, ev.Normal)
	}
}

func fireEvent(name string, params ev.M, key string) {
	if obj := getEvManager(key, false); obj != nil {
		obj.Fire(name, params)
	}
}

func asyncFireEvent(name string, params ev.M, key string) {
	go func() {
		fireEvent(name, params, key)
	}()
}
