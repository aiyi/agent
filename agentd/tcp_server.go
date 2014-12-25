package agentd

import (
	"net"
	"runtime"
	"strings"
)

func tcpServer(ad *AGENTD) {
	listener := ad.tcpListener
	ad.logf("TCP: listening on %s", listener.Addr())

	for {
		clientConn, err := listener.Accept()
		if err != nil {
			if nerr, ok := err.(net.Error); ok && nerr.Temporary() {
				ad.logf("NOTICE: temporary Accept() failure - %s", err)
				runtime.Gosched()
				continue
			}
			// theres no direct way to detect this error because it is not exposed
			if !strings.Contains(err.Error(), "use of closed network connection") {
				ad.logf("ERROR: listener.Accept() - %s", err)
			}
			break
		}
		ad.logf("TCP: new client(%s)", clientConn.RemoteAddr())
		go NewConn(clientConn).Start()
	}

	ad.logf("TCP: closing %s", listener.Addr())
}
