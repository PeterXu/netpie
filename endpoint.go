package main

import (
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
	cc       *ShellCompleter
}

func (e *Endpoint) Init(sigaddr string) {
	e.signal = NewSignalClient()
	e.signal.sigaddr = sigaddr

	// listen remote-peer's events
	events := []string{
		kActionEventIceOpen,
		kActionEventIceClose,
		kActionEventIceOpenAck,
		kActionEventIceCloseAck,
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
		e.CheckOpenLocalService("ev_open", resp.ServiceName, resp.FromId)

		req := NewSignalRequest(e.signal.id)
		req.ToId = resp.FromId
		req.ServiceName = resp.ServiceName
		e.signal.SendRequest(kActionEventIceOpenAck, req)
	case kActionEventIceClose:
		e.CheckOpenLocalService("ev_close", resp.ServiceName, resp.FromId)

		req := NewSignalRequest(e.signal.id)
		req.ToId = resp.FromId
		req.ServiceName = resp.ServiceName
		e.signal.SendRequest(kActionEventIceCloseAck, req)
	case kActionEventIceOpenAck:
		e.CheckOpenLocalService("ev_openack", resp.ServiceName, resp.FromId)
	case kActionEventIceCloseAck:
		e.CheckOpenLocalService("ev_closeack", resp.ServiceName, resp.FromId)
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

func (e *Endpoint) CheckEnableLocalService(action, name string) (err error) {
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
	}
	return
}

func (e *Endpoint) CheckOpenLocalService(action, name, fromId string) (err error) {
	switch action {
	case "ev_open", "ev_openack":
		if db, ok := e.services[name]; ok {
			srvId := e.GetLocalServiceKey(fromId)
			if item, ok := db.items[srvId]; !ok {
				// service provider should start client-mode
				// service requester should start server-mode
				isServiceProvider := (action == "ev_open")
				item = NewLocalService(name, !isServiceProvider)
				//TODO
				db.items[srvId] = item
			}
		}
	case "ev_close", "ev_closeack":
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
	e.cc = cc
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

	var err error
	var parts []string

	if parts, err = ParseCommandLine(line); err != nil {
		fmt.Println("error: ", err)
		return
	}

	if len(parts[0]) == 0 {
		return
	}

	if !e.cc.IsExist(parts[0]) {
		fmt.Printf("warn: %s not exist\n", parts[0])
		return
	} else {
		switch parts[0] {
		case "help":
			e.cc.PrintHelp()
			return
		}
	}

	// do PreRun if exist
	if e.hook != nil {
		if err = e.hook.PreRunSignal(parts); err != nil {
			return
		}
	}

	// do Run
	var ret *Result
	if ret, err = e.GoRun(parts[0], parts[1:]); err != nil {
		fmt.Printf("== %s failed: %v\n", parts[0], err)
	} else {
		fmt.Printf("== %s success\n", parts[0])
	}
	if ret != nil {
		fmt.Println("== result: \n", ret.data)
	}
	//fmt.Println(":", line, len(parts), parts, err)

	// do PostRun if exist
	if e.hook != nil {
		e.hook.PostRunSignal(parts, err)
	}
}

func (e *Endpoint) GoRun(action string, params []string) (*Result, error) {
	if fn, ok := e.signal.actions[action]; ok {
		return fn(action, params)
	} else {
		return nil, errFnInvalidAction(action)
	}
}
