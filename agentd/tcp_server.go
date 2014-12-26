package agentd

import (
	"net"
	"runtime"
	"strings"
)

func tcpServer(a *AgentD) {
	listener := a.tcpListener
	a.logf("TCP: listening on %s", listener.Addr())

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				a.logf("NOTICE: temporary Accept() failure - %s", err)
				runtime.Gosched()
				continue
			}
			// theres no direct way to detect this error because it is not exposed
			if !strings.Contains(err.Error(), "use of closed network connection") {
				a.logf("ERROR: listener.Accept() - %s", err)
			}
			break
		}
		a.logf("TCP: new client(%s)", clientConn.RemoteAddr())
		go NewConn(a, clientConn).Start()
	}

	a.logf("TCP: closing %s", listener.Addr())
}
