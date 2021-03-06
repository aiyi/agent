package main

import (
	"flag"
	"fmt"
	"github.com/aiyi/agent/agent"
	"github.com/aiyi/agent/rsu"
	"github.com/aiyi/agent/util"
	"github.com/mreiferson/go-options"
	"os"
	"os/signal"
	"syscall"
)

var (
	flagset = flag.NewFlagSet("agentd", flag.ExitOnError)

	showVersion = flagset.Bool("version", false, "print version string")
	tcpAddress  = flagset.String("tcp-address", "0.0.0.0:3002", "<addr>:<port> to listen on for TCP clients")
)

func main() {
	flagset.Parse(os.Args[1:])

	if *showVersion {
		fmt.Println(util.Version("agentd"))
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	opts := agent.NewAgentdOptions()
	options.Resolve(opts, flagset, nil)

	a := agent.NewAgentD(opts, &rsu.RsuProtocol{})
	r := rsu.NewRestServer(a)
	
	r.Main()
	a.Main()

	<-signalChan
	a.Exit()
	r.Exit()
}
