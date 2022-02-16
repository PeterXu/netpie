package main

import "C"
import (
	"flag"
	"fmt"
	"os"
)

func main() {
	var client_signal_addr string
	clientFlags := flag.NewFlagSet("client", flag.ExitOnError)
	clientFlags.StringVar(&client_signal_addr, "sigaddr", "127.0.0.1:9527", "The address of signal server")

	var server_signal_addr string
	var server_listen_addr string
	serverFlags := flag.NewFlagSet("server", flag.ExitOnError)
	serverFlags.StringVar(&server_signal_addr, "sigaddr", "127.0.0.1:9527", "The address of signal server")
	serverFlags.StringVar(&server_listen_addr, "addr", "0.0.0.0:9090", "The address of server listen")

	var signal_listen_addr string
	signalFlags := flag.NewFlagSet("signal", flag.ExitOnError)
	signalFlags.StringVar(&signal_listen_addr, "addr", "0.0.0.0:9527", "The address of signal listen")

	usage := func() {
		fmt.Printf("usage: %s command\n", os.Args[0])
		fmt.Println("client")
		clientFlags.PrintDefaults()
		fmt.Println("server")
		serverFlags.PrintDefaults()
		fmt.Println("signal")
		signalFlags.PrintDefaults()
	}

	if len(os.Args) < 2 {
		usage()
		os.Exit(1)
	}

	ch := make(chan bool)
	defer func() {
		close(ch)
	}()

	switch os.Args[1] {
	case "client":
		clientFlags.Parse(os.Args[2:])
		fmt.Println(client_signal_addr)
		client := NewClient(client_signal_addr)
		client.StartShell()
	case "server":
		serverFlags.Parse(os.Args[2:])
		fmt.Println(server_signal_addr, server_listen_addr)
		server := NewServer(server_signal_addr)
		server.StartShell()
	case "signal":
		signalFlags.Parse(os.Args[2:])
		fmt.Println(signal_listen_addr)
		signal := NewSignalServer()
		signal.Start(signal_listen_addr)
	default:
		usage()
		os.Exit(1)
	}
}
