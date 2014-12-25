package agentd

import (
	"agent/util"
	"fmt"
	"log"
	"net"
	"os"
)

type AGENTD struct {
	// 64bit atomic vars need to be first for proper alignment on 32bit platforms
	clientIDSequence int64

	opts *AgentdOptions

	tcpAddr     *net.TCPAddr
	tcpListener net.Listener

	notifyChan chan interface{}
	exitChan   chan int
	waitGroup  util.WaitGroupWrapper

	logger *log.Logger
}

func NewAGENTD(opts *AgentdOptions) *AGENTD {
	ad := &AGENTD{
		opts:       opts,
		exitChan:   make(chan int),
		notifyChan: make(chan interface{}),
		logger:     log.New(os.Stderr, "[agentd] ", log.Ldate|log.Ltime|log.Lmicroseconds),
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", opts.TcpAddress)
	if err != nil {
		ad.logf("FATAL: failed to resolve TCP address (%s) - %s", opts.TcpAddress, err)
		os.Exit(1)
	}
	ad.tcpAddr = tcpAddr

	return ad
}

func (ad *AGENTD) Main() {
	tcpListener, err := net.Listen("tcp", ad.tcpAddr.String())
	if err != nil {
		ad.logf("FATAL: listen (%s) failed - %s", ad.tcpAddr, err)
		os.Exit(1)
	}
	ad.tcpListener = tcpListener

	ad.waitGroup.Wrap(func() {
		tcpServer(ad)
	})
}

func (ad *AGENTD) Exit() {
	if ad.tcpListener != nil {
		ad.tcpListener.Close()
	}

	// we want to do this last as it closes the idPump (if closed first it
	// could potentially starve items in process and deadlock)
	close(ad.exitChan)
	ad.waitGroup.Wait()
}

func (ad *AGENTD) logf(f string, args ...interface{}) {
	if ad.logger == nil {
		return
	}
	ad.logger.Output(2, fmt.Sprintf(f, args...))
}
