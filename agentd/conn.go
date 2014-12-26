package agentd

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// LogLevel specifies the severity of a given log message
type LogLevel int

type logger interface {
	Output(calldepth int, s string) error
}

// logging constants
const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError

	LogLevelDebugPrefix   = "DBG"
	LogLevelInfoPrefix    = "INF"
	LogLevelWarningPrefix = "WRN"
	LogLevelErrorPrefix   = "ERR"
)

// LogPrefix Resolution
func logPrefix(lvl LogLevel) string {
	var prefix string

	switch lvl {
	case LogLevelDebug:
		prefix = LogLevelDebugPrefix
	case LogLevelInfo:
		prefix = LogLevelInfoPrefix
	case LogLevelWarning:
		prefix = LogLevelWarningPrefix
	case LogLevelError:
		prefix = LogLevelErrorPrefix
	}

	return prefix
}

// ProducerTransaction is returned by the async publish methods
// to retrieve metadata about the command after the
// response is received.
type ProducerTransaction struct {
	cmd      *Command
	doneChan chan *ProducerTransaction
	resp     Message
}

func (t *ProducerTransaction) finish() {
	if t.doneChan != nil {
		t.doneChan <- t
	}
}

// Conn represents a connection to nsqd
//
// Conn exposes a set of callbacks for the
// various events that occur on a connection
type Conn struct {
	mtx sync.Mutex

	agentd *AgentD

	addr string
	conn *net.TCPConn

	protocol Protocol // data protocol

	logger logger
	logLvl LogLevel
	logFmt string

	r io.Reader
	w io.Writer

	transactionChan chan *ProducerTransaction
	msgResponseChan chan Message
	exitChan        chan int
	drainReady      chan int

	transactions        []*ProducerTransaction
	concurrentProducers int32

	closeFlag int32
	stopper   sync.Once
	wg        sync.WaitGroup

	readLoopRunning int32
}

// NewConn returns a new Conn instance
func NewConn(a *AgentD, conn net.Conn) *Conn {
	return &Conn{
		agentd: a,
		addr:   conn.RemoteAddr().String(),
		conn:   conn.(*net.TCPConn),
		r:      bufio.NewReader(conn),
		w:      bufio.NewWriter(conn),

		protocol: &LtvProtocol{},

		logger: log.New(os.Stderr, "", log.Flags()),
		logLvl: LogLevelInfo,

		transactionChan: make(chan *ProducerTransaction),
		msgResponseChan: make(chan Message),
		exitChan:        make(chan int),
		drainReady:      make(chan int),
	}
}

// SetLogger assigns the logger to use as well as a level.
//
// The logger parameter is an interface that requires the following
// method to be implemented (such as the the stdlib log.Logger):
//
//    Output(calldepth int, s string)
//
func (c *Conn) SetLogger(l logger, lvl LogLevel) {
	c.logger = l
	c.logLvl = lvl
}

func (c *Conn) Start() {
	c.log(LogLevelInfo, "client connected")
	c.wg.Add(2)
	atomic.StoreInt32(&c.readLoopRunning, 1)
	go c.readLoop()
	go c.writeLoop()
	c.agentd.AddClient(strings.Split(c.addr, ":")[0], c)
}

// Close idempotently initiates connection close
func (c *Conn) Close() error {
	atomic.StoreInt32(&c.closeFlag, 1)
	return nil
}

// IsClosing indicates whether or not the
// connection is currently in the processing of
// gracefully closing
func (c *Conn) IsClosing() bool {
	return atomic.LoadInt32(&c.closeFlag) == 1
}

// String returns the fully-qualified address
func (c *Conn) String() string {
	return c.addr
}

// Read performs a deadlined read on the underlying TCP connection
func (c *Conn) Read(p []byte) (int, error) {
	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	return c.r.Read(p)
}

// Write performs a deadlined write on the underlying TCP connection
func (c *Conn) Write(p []byte) (int, error) {
	c.conn.SetWriteDeadline(time.Now().Add(1 * time.Second))
	return c.w.Write(p)
}

// WriteCommand is a goroutine safe method to write a Command
// to this connection, and flush.
func (c *Conn) WriteCommand(cmd *Command) error {
	c.mtx.Lock()

	_, err := cmd.WriteTo(c)
	if err != nil {
		goto exit
	}
	err = c.Flush()

exit:
	c.mtx.Unlock()
	if err != nil {
		c.log(LogLevelError, "IO error - %s", err)
	}
	return err
}

// WriteMessage is a goroutine safe method to write a Message
// to this connection, and flush.
func (c *Conn) WriteMessage(msg Message) error {
	c.mtx.Lock()

	err := c.protocol.WriteMessage(c, msg)
	if err != nil {
		goto exit
	}
	err = c.Flush()

exit:
	c.mtx.Unlock()
	if err != nil {
		c.log(LogLevelError, "IO error - %s", err)
	}
	return err
}

type flusher interface {
	Flush() error
}

// Flush writes all buffered data to the underlying TCP connection
func (c *Conn) Flush() error {
	if f, ok := c.w.(flusher); ok {
		return f.Flush()
	}
	return nil
}

func (c *Conn) SendCommand(cmd *Command) (Message, error) {
	atomic.AddInt32(&c.concurrentProducers, 1)

	if atomic.LoadInt32(&c.closeFlag) == 1 {
		atomic.AddInt32(&c.concurrentProducers, -1)
		return nil, ErrNotConnected
	}

	doneChan := make(chan *ProducerTransaction)
	trans := &ProducerTransaction{
		cmd:      cmd,
		doneChan: doneChan,
	}

	c.transactionChan <- trans
	atomic.AddInt32(&c.concurrentProducers, -1)

	t := <-doneChan
	return t.resp, nil
}

func (c *Conn) popTransaction(frameType int32, resp Message) {
	t := c.transactions[0]
	c.transactions = c.transactions[1:]

	t.resp = resp
	t.finish()
}

func (c *Conn) readLoop() {
	for {
		if atomic.LoadInt32(&c.closeFlag) == 1 {
			goto exit
		}

		frameType, msg, err := c.protocol.DecodeMessage(c)
		if err != nil {
			if !strings.Contains(err.Error(), "use of closed network connection") {
				c.log(LogLevelError, "IO error - %s", err)
			}
			goto exit
		}

		switch frameType {
		case FrameTypeMessage:
			resp := c.protocol.HandleMessage(msg)
			if resp != nil {
				c.msgResponseChan <- resp
			}
		case FrameTypeResponse:
			c.popTransaction(FrameTypeResponse, msg)
		default:
			c.log(LogLevelError, "IO error - %s", err)
			c.log(LogLevelError, "unknown frame type %d", frameType)
			goto exit
		}
	}

exit:
	atomic.StoreInt32(&c.readLoopRunning, 0)
	// start the connection close
	c.close()
	c.wg.Done()
	c.log(LogLevelInfo, "readLoop exiting")
}

func (c *Conn) writeLoop() {
	for {
		select {
		case <-c.exitChan:
			c.log(LogLevelInfo, "breaking out of writeLoop")
			// Indicate drainReady because we will not pull any more off msgResponseChan
			close(c.drainReady)
			goto exit
		case t := <-c.transactionChan:
			c.transactions = append(c.transactions, t)
			err := c.WriteCommand(t.cmd)
			if err != nil {
				c.log(LogLevelError, "error sending command %s - %s", t.cmd, err)
				c.close()
				continue
			}
		case resp := <-c.msgResponseChan:
			err := c.WriteMessage(resp)
			if err != nil {
				c.log(LogLevelError, "error sending response %s - %s", resp, err)
				c.close()
				continue
			}
		}
	}

exit:
	c.wg.Done()
	c.log(LogLevelInfo, "writeLoop exiting")
}

func (c *Conn) close() {
	// a "clean" connection close is orchestrated as follows:
	//
	//     3. set c.closeFlag
	//     4. readLoop() exits
	//         call close()
	//     5. c.exitChan close
	//         a. writeLoop() exits
	//             i. c.drainReady close
	//     6a. launch cleanup() goroutine (we're racing with intraprocess
	//        routed messages, see comments below)
	//         a. wait on c.drainReady
	//         b. loop and receive on c.msgResponseChan chan
	//            until messages-in-flight == 0
	//            i. ensure that readLoop has exited
	//     6b. launch waitForCleanup() goroutine
	//         b. wait on waitgroup (covers readLoop() and writeLoop()
	//            and cleanup goroutine)
	//         c. underlying TCP connection close
	//
	c.agentd.RemoveClient(strings.Split(c.addr, ":")[0])

	c.stopper.Do(func() {
		c.log(LogLevelInfo, "beginning close")
		atomic.StoreInt32(&c.closeFlag, 1)
		close(c.exitChan)
		c.conn.CloseRead()

		c.wg.Add(1)
		go c.cleanup()

		go c.waitForCleanup()
	})
}

func (c *Conn) cleanup() {
	<-c.drainReady
	// writeLoop has exited, drain any remaining in flight messages
	for {
		select {
		case <-c.msgResponseChan:
		default:
			// until the readLoop has exited we cannot be sure that there
			// still won't be a race
			if atomic.LoadInt32(&c.readLoopRunning) == 0 {
				goto exit
			}
			// give the runtime a chance to schedule other racing goroutines
			time.Sleep(5 * time.Millisecond)
			continue
		}
	}

exit:
	c.transactionCleanup()
	c.wg.Done()
	c.log(LogLevelInfo, "finished draining, cleanup exiting")
}

func (c *Conn) transactionCleanup() {
	// clean up transactions we can easily account for
	for _, t := range c.transactions {
		t.resp = nil
		t.finish()
	}
	c.transactions = c.transactions[:0]

	// spin and free up any writes that might have raced
	// with the cleanup process (blocked on writing
	// to transactionChan)
	for {
		select {
		case t := <-c.transactionChan:
			t.resp = nil
			t.finish()
		default:
			// keep spinning until there are 0 concurrent producers
			if atomic.LoadInt32(&c.concurrentProducers) == 0 {
				return
			}
			// give the runtime a chance to schedule other racing goroutines
			time.Sleep(5 * time.Millisecond)
			continue
		}
	}
}

func (c *Conn) waitForCleanup() {
	// this blocks until readLoop and writeLoop
	// (and cleanup goroutine above) have exited
	c.wg.Wait()
	//c.conn.CloseWrite()
	c.conn.Close()
	c.log(LogLevelInfo, "clean close complete")
}

func (c *Conn) log(lvl LogLevel, line string, args ...interface{}) {
	if c.logger == nil {
		return
	}

	if c.logLvl > lvl {
		return
	}

	c.logger.Output(2, fmt.Sprintf("%-4s %s %s", logPrefix(lvl),
		fmt.Sprintf("(%s)", c.String()),
		fmt.Sprintf(line, args...)))
}
