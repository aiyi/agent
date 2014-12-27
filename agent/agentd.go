package agent

import (
	"fmt"
	"github.com/aiyi/agent/util"
	"log"
	"net"
	"os"
	"strings"
	"sync"
)

type AgentD struct {
	// 64bit atomic vars need to be first for proper alignment on 32bit platforms
	clientIDSequence int64

	sync.RWMutex

	opts     *AgentdOptions
	protocol Protocol

	tcpAddr     *net.TCPAddr
	tcpListener net.Listener

	clients map[string]*Conn

	notifyChan chan interface{}
	exitChan   chan int
	waitGroup  util.WaitGroupWrapper

	logger *log.Logger
}

func NewAgentD(opts *AgentdOptions, proto Protocol) *AgentD {
	a := &AgentD{
		opts:       opts,
		protocol:   proto,
		clients:    make(map[string]*Conn),
		exitChan:   make(chan int),
		notifyChan: make(chan interface{}),
		logger:     log.New(os.Stderr, "[agentd] ", log.Ldate|log.Ltime|log.Lmicroseconds),
	}

	tcpAddr, err := net.ResolveTCPAddr("tcp", opts.TcpAddress)
	if err != nil {
		a.logf("FATAL: failed to resolve TCP address (%s) - %s", opts.TcpAddress, err)
		os.Exit(1)
	}
	a.tcpAddr = tcpAddr

	return a
}

func (a *AgentD) GetServerIP() string {
	return strings.Split(a.tcpAddr.String(), ":")[0]
}

func (a *AgentD) AddClient(clientID string, client *Conn) {
	a.Lock()
	defer a.Unlock()

	_, ok := a.clients[clientID]
	if ok {
		return
	}
	a.clients[clientID] = client
}

func (a *AgentD) RemoveClient(clientID string) {
	a.Lock()
	defer a.Unlock()

	_, ok := a.clients[clientID]
	if !ok {
		return
	}
	delete(a.clients, clientID)
}

func (a *AgentD) GetClient(clientID string) (*Conn, bool) {
	a.RLock()
	defer a.RUnlock()

	c, ok := a.clients[clientID]
	if !ok {
		return nil, false
	}
	return c, true
}

func (a *AgentD) Main() {
	tcpListener, err := net.Listen("tcp", a.tcpAddr.String())
	if err != nil {
		a.logf("FATAL: listen (%s) failed - %s", a.tcpAddr, err)
		os.Exit(1)
	}
	a.tcpListener = tcpListener

	a.waitGroup.Wrap(func() {
		tcpServer(a)
	})
}

func (a *AgentD) Exit() {
	if a.tcpListener != nil {
		a.tcpListener.Close()
	}

	// we want to do this last as it closes the idPump (if closed first it
	// could potentially starve items in process and deadlock)
	close(a.exitChan)
	a.waitGroup.Wait()
}

func (a *AgentD) logf(f string, args ...interface{}) {
	if a.logger == nil {
		return
	}
	a.logger.Output(2, fmt.Sprintf(f, args...))
}
