package main

import (
	ev "github.com/gookit/event"
)

type evManager = ev.Manager
type evEvent = ev.Event
type evData = ev.M

/// ev objects

type EvListenerFunc func(e evEvent) error

type EvObject struct {
	*evManager
	listeners map[string]EvListenerFunc
}

func NewEvObject() *EvObject {
	return &EvObject{
		evManager: ev.NewManager("event"),
		listeners: make(map[string]EvListenerFunc),
	}
}

func (obj *EvObject) Handle(e evEvent) error {
	if fn, ok := obj.listeners[e.Name()]; ok {
		return fn(e)
	} else {
		return nil
	}
}

func (obj *EvObject) ListenEvents(names []string, listener EvListenerFunc) {
	for _, name := range names {
		obj.ListenEvent(name, listener)
	}
}

// only listen once with the same key
func (obj *EvObject) ListenEvent(name string, listener EvListenerFunc) {
	if _, ok := obj.listeners[name]; !ok {
		obj.Listen(name, obj, ev.Normal)
	}
	obj.listeners[name] = listener
}

func (obj *EvObject) FireEvent(name string, params ev.M) {
	obj.Fire(name, params)
}

func (obj *EvObject) AsyncFireEvent(name string, params ev.M) {
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
