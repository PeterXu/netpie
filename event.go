package main

import (
	ev "github.com/gookit/event"
)

type evManager = ev.Manager
type evEvent = ev.Event
type evData = ev.M

/// ev objects

type EvObject struct {
	*ev.Manager
}

func newEvObject() *EvObject {
	return &EvObject{ev.NewManager("event")}
}

func (obj *EvObject) listenEvent(name string, listener ev.Listener) {
	obj.Listen(name, listener, ev.Normal)
}

func (obj *EvObject) fireEvent(name string, params ev.M) {
	obj.Fire(name, params)
}

func (obj *EvObject) asyncFireEvent(name string, params ev.M) {
	go func() {
		obj.Fire(name, params)
	}()
}

/// ev managers

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

func delEvManager(key string) {
	delete(gEvManagers, key)
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
