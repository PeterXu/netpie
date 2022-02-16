package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	util "github.com/PeterXu/goutil"
	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

type EndpointHook interface {
	PreRunSignal(params []string) error
	PostRunSignal(params []string, err error)
}

func NewLocalServiceDB() *LocalServiceDB {
	return &LocalServiceDB{
		items: make(map[string]*LocalService),
	}
}

type LocalServiceDB struct {
	items map[string]*LocalService // key: myid@peerid
}

func NewEndpoint(hook EndpointHook, isServer bool) *Endpoint {
	return &Endpoint{
		hook:     hook,
		isServer: isServer,
		services: make(map[string]*LocalServiceDB),
	}
}

type Endpoint struct {
	hook     EndpointHook
	isServer bool
	services map[string]*LocalServiceDB // key: serviceName
	signal   *SignalClient
}

func (e *Endpoint) Init(sigaddr string) {
	e.signal = NewSignalClient()
	e.signal.sigaddr = sigaddr

	// listen remote-peer's events
	events := []string{
		kActionEventIceOpen,
		kActionEventIceClose,
		kActionEventIceAuth,
		kActionEventIceCandidate,
	}
	e.signal.ListenEvents(events, func(ev evEvent) error {
		if resp := ev.Get("data").(*SignalResponse); resp != nil {
			e.OnRemoteEvent(resp)
		}
		return nil
	})
}

func (e *Endpoint) OnRemoteEvent(resp *SignalResponse) error {
	switch resp.Event {
	case kActionEventIceOpen:
		e.ControlLocalService("open", resp.ServiceName, resp.FromId)
		//e.signal.ch_send <- resp
	case kActionEventIceClose:
		e.ControlLocalService("close", resp.ServiceName, resp.FromId)
		resp.Event = kActionEventIceCloseAck
	case kActionEventIceAuth:
		if srv := e.GetLocalService(resp.ServiceName, resp.FromId); srv != nil {
			srv.OnIceAuth(resp.ResultM["ice-ufrag"], resp.ResultM["ice-pwd"])
		}
	case kActionEventIceCandidate:
		if srv := e.GetLocalService(resp.ServiceName, resp.FromId); srv != nil {
			srv.OnIceCandidate(resp.ResultM["ice-candidate"])
		}
	}
	return nil
}

func (e *Endpoint) GetLocalServiceKey(fromId string) string {
	return e.signal.id + "@" + fromId
}

func (e *Endpoint) GetLocalService(name, fromId string) *LocalService {
	if db, ok := e.services[name]; ok {
		srvId := e.GetLocalServiceKey(fromId)
		if item, ok := db.items[srvId]; ok {
			return item
		}
	}
	return nil
}

func (e *Endpoint) ControlLocalService(action, name, fromId string) (err error) {
	switch action {
	case "enable":
		if _, ok := e.services[name]; !ok {
			e.services[name] = NewLocalServiceDB()
		}
	case "disable":
		if db, ok := e.services[name]; ok {
			for _, item := range db.items {
				item.Uninit()
			}
			delete(e.services, name)
		}
	case "open":
		if db, ok := e.services[name]; ok {
			srvId := e.GetLocalServiceKey(fromId)
			if item, ok := db.items[srvId]; !ok {
				isLocalServer := !e.isServer
				item = NewLocalService(name, isLocalServer)
				db.items[srvId] = item
			}
		}
	case "close":
		if db, ok := e.services[name]; ok {
			srvId := e.GetLocalServiceKey(fromId)
			if item, ok := db.items[srvId]; !ok {
				item.Uninit()
				delete(db.items, srvId)
			}
		}
	}
	return
}

func (e *Endpoint) StartShell(title string) {
	fmt.Println("Please use `exit` or `Ctrl-D` to exit this program.")
	defer fmt.Println("Bye!")
	defer util.HandleTTYOnExit()

	cc := NewShellCompleter()
	cc.Init(e.isServer)
	p := prompt.New(
		e.Executor,
		cc.Complete,
		prompt.OptionTitle(fmt.Sprintf("%s: interactive cmdline", title)),
		prompt.OptionPrefix(">>> "),
		prompt.OptionInputTextColor(prompt.Blue),
		prompt.OptionCompletionWordSeparator(completer.FilePathCompletionSeparator),
	)
	p.Run()
}

func (e *Endpoint) Executor(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	} else if line == "quit" || line == "exit" {
		fmt.Println("Bye!")
		os.Exit(0)
		return
	}

	var parts []string
	for _, item := range strings.Split(line, " ") {
		parts = append(parts, strings.Trim(item, "\""))
	}

	// do PreRun if exist
	if e.hook != nil {
		if err := e.hook.PreRunSignal(parts); err != nil {
			return
		}
	}

	// do Run
	err := e.GoRun(parts[0], parts[1:])
	fmt.Println(":", line, parts, err)

	// do PostRun if exist
	if e.hook != nil {
		e.hook.PostRunSignal(parts, err)
	}
}

func (e *Endpoint) GoRun(action string, params []string) error {
	if fn, ok := e.signal.actions[action]; ok {
		return fn(action, params)
	} else {
		return errors.New("invalid action:" + action)
	}
}
