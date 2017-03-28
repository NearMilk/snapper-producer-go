package snapper

import (
	"errors"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/teambition/jsonrpc-go"
)

const retryDelay = 500 * time.Millisecond

// Connector for connect the remote server
type connector struct {
	addr              string
	socket            *socket
	notificationQueue *Queue
	rpcs              *rpcmap
	closed            chan bool
	rpctimeout        time.Duration
	id                uint64
	queue             chan string
	onSign            getSignature
	onReconnect       reconnecting
	onError           erroring
	connected         int32
}

type getSignature func() string
type reconnecting func(error)
type erroring func(error)
type closing func(error)

func newConnector(address string, rpctimeouts ...time.Duration) (conn *connector) {
	conn = &connector{
		addr:              address,
		queue:             make(chan string, 10000),
		notificationQueue: NewQueue(),
		rpcs:              newrpcmap(),
		closed:            make(chan bool),
	}
	if len(rpctimeouts) > 0 && rpctimeouts[0] > time.Second {
		conn.rpctimeout = rpctimeouts[0]
	} else {
		conn.rpctimeout = 40 * time.Second
	}
	return
}
func (conn *connector) start() (err error) {
	conn.socket, err = conn.createConn()
	if err == nil {
		conn.setConnected(true)
		go conn.write()
		go conn.read()
	}
	return
}
func (conn *connector) write() {
	var content string
	for {
		select {
		case <-conn.closed:
			return
		default:
			if content == "" {
				content = <-conn.queue
			}
			_, err := conn.socket.WriteBulkString(content)
			if err != nil {
				conn.sendError(err)
				conn.socket.Close()
				conn.reconnect()
			} else {
				content = ""
			}
		}
	}
}
func (conn *connector) read() {
	for {

		result, err := conn.socket.ReadString()
		if err != nil {
			conn.sendError(err)
			return
		}
		res, _ := jsonrpc.Parse(result)
		if res.Type == jsonrpc.InvalidType {
			conn.sendError(errors.New(res.Error.Message))
			continue
		}
		id, _ := res.ID.(string)
		rpc, ok := conn.rpcs.get(id)
		if ok {
			if res.Error != nil {
				err = errors.New(res.Error.Message)
				rpc.callback <- &resultItem{result: nil, err: err}
			} else {
				rpc.callback <- &resultItem{result: res.Result, err: nil}
			}
			conn.rpcs.delete(id)
		}
	}
}

// Request sending a jsonrpc Request object to a Server.
func (conn *connector) request(method string, args interface{}) (result interface{}, err error) {
	if !conn.isConnected() {
		return nil, errors.New("socket was closed")
	}
	requestid := conn.randID()
	item := &requestItem{expiretime: time.Now(), callback: make(chan *resultItem)}
	conn.rpcs.set(requestid, item)
	value, _ := jsonrpc.Request(requestid, method, args)
	conn.queue <- value
	timeout := time.NewTimer(conn.rpctimeout)
	select {
	case msg := <-item.callback:
		result = msg.result
		err = msg.err
	case <-timeout.C:
		err = errors.New("Send RPC time out," + requestid)
	}
	timeout.Stop()
	conn.rpcs.delete(requestid)
	close(item.callback)
	return
}

// Notification sending a jsonrpc Notification to server without an "id" member.The server not reply this message.
func (conn *connector) notification(args []string) {
	if !conn.isConnected() || len(conn.queue) > 0 {
		conn.notificationQueue.Push(args)
	}
	var arrays []interface{}
	if conn.notificationQueue.Len() > 0 {
		arrays = conn.notificationQueue.PopN(50)
	} else {
		arrays = append(arrays, args)
	}
	rpc, _ := jsonrpc.Notification("publish", arrays)
	conn.queue <- rpc
}

func (conn *connector) close() {
	close(conn.closed)
	conn.socket.Close()
}
func (conn *connector) reconnect() {
	conn.setConnected(false)
	for {
		time.Sleep(retryDelay)
		select {
		case <-conn.closed:
			return
		default:
			var err error
			conn.socket, err = conn.createConn()
			if conn.onReconnect != nil {
				go conn.onReconnect(err)
			}
			if err == nil {
				go conn.read()
				conn.setConnected(true)
				return
			}
		}
	}
}
func (conn *connector) createConn() (*socket, error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", conn.addr)
	if err != nil {
		return nil, err
	}
	tcp, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	tcp.SetKeepAlive(true)
	socket := newSocket(tcp)
	if conn.onSign != nil {
		reply, autherr := conn.auth(socket, conn.onSign())
		if autherr != nil {
			socket.Close()
			return nil, errors.New(reply)
		}
		if rpcerr := conn.validateAuth(reply); rpcerr != nil {
			return nil, rpcerr
		}
	}
	return socket, nil
}

// Auth for get authorization from server with sign
func (conn *connector) auth(socket *socket, sign string) (reply string, err error) {
	_, err = socket.WriteBulkString(sign, time.Second*30)
	if err != nil {
		return
	}
	reply, err = socket.ReadString(time.Second * 30)
	return
}
func (conn *connector) validateAuth(reply string) error {
	res, _ := jsonrpc.Parse(reply)
	if res.Error != nil {
		return errors.New(res.Error.Message)
	}
	return nil
}

// randID generates a request ID
func (conn *connector) randID() string {
	ISN := strconv.FormatInt(time.Now().UnixNano(), 36)
	id := atomic.AddUint64(&conn.id, 1)
	return ISN + ":" + strconv.FormatUint(id, 36)
}

func (conn *connector) sendError(err error) {
	if conn.onError != nil {
		go conn.onError(err)
	}
}

func (conn *connector) setConnected(b bool) {
	var connected int32
	if b {
		connected = 1
	} else {
		connected = 0
	}
	atomic.StoreInt32(&conn.connected, connected)
}
func (conn *connector) isConnected() bool {
	return atomic.LoadInt32(&conn.connected) == 1
}
