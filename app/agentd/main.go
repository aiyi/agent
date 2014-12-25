package main

import (
	"agent/agentd"
	"agent/util"
	"flag"
	"fmt"
	"github.com/mreiferson/go-options"
	"os"
	"os/signal"
	"syscall"
)

var (
	flagset = flag.NewFlagSet("agentd", flag.ExitOnError)

	showVersion = flagset.Bool("version", false, "print version string")
	tcpAddress  = flagset.String("tcp-address", "0.0.0.0:4050", "<addr>:<port> to listen on for TCP clients")
)

func main() {
	fmt.Println("Hello World!")
	flagset.Parse(os.Args[1:])

	if *showVersion {
		fmt.Println(util.Version("agentd"))
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	opts := agentd.NewAgentdOptions()
	options.Resolve(opts, flagset, nil)
	ad := agentd.NewAGENTD(opts)

	ad.Main()
	<-signalChan
	ad.Exit()
}
