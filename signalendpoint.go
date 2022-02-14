package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/c-bata/go-prompt/completer"
)

type SignalEndpointHook interface {
	PreRunSignal(params []string) error
	PostRunSignal(params []string, err error)
}

func NewSignalEndpoint(hook SignalEndpointHook) *SignalEndpoint {
	return &SignalEndpoint{
		hook: hook,
	}
}

type SignalEndpoint struct {
	hook   SignalEndpointHook
	signal *SignalClient
}

func (e *SignalEndpoint) Init(sigaddr string, cb SignalEventCallback) {
	e.signal = NewSignalClient(cb)
	e.signal.sigaddr = sigaddr
}

func (e *SignalEndpoint) StartShell(title string) {
	fmt.Println("Please use `exit` or `Ctrl-D` to exit this program.")
	defer fmt.Println("Bye!")
	defer HandleTTYOnExit()

	cc := newShellCompleter()
	cc.Init(title)
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

func (e *SignalEndpoint) Executor(line string) {
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

func (e *SignalEndpoint) GoRun(cmd string, params []string) error {
	if fn, ok := e.signal.actions[cmd]; ok {
		return fn(params)
	} else {
		return errors.New("invalid command:" + cmd)
	}
}
