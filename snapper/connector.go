package snapper

import (
	"errors"
	"net"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/teambition/jsonrpc-go"
	"github.com/teambition/snapper-producer-go/snapper/service"
	"github.com/teambition/snapper-producer-go/snapper/util"
)

var (
	// ReconnectEvent ...
	ReconnectEvent = "reconnect"
	// ErrorEvent ...
	ErrorEvent = "error"
	// CloseEvent ...
	CloseEvent = "close"
)

// Event ...
type Event struct {
	EventType string
	EventData error
}
type parent interface {
	getSignature() string
	sendEvent(ev *Event)
}

const retryDelay = 500 * time.Millisecond

// Connector for connect the remote server
type connector struct {
	addr              string
	socket            *service.Socket
	notificationQueue *util.Queue
	rpcs              *util.RPCMap
	closed            chan bool
	rpctimeout        time.Duration
	id                uint64
	queue             chan []byte
	connected         int32
	p                 parent
}

func newConnector(address string, p parent, rpctimeouts ...time.Duration) (conn *connector) {
	conn = &connector{
		addr:              address,
		queue:             make(chan []byte, 10000),
		notificationQueue: util.NewQueue(),
		rpcs:              util.NewRPCMap(),
		closed:            make(chan bool),
		p:                 p,
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
	var content []byte
	for {
		select {
		case <-conn.closed:
			return
		default:
			if content == nil {
				content = <-conn.queue
			}
			_, err := conn.socket.WriteBulkString(content)
			if err != nil {
				conn.p.sendEvent(&Event{EventType: ErrorEvent, EventData: err})
				conn.socket.Close()
				conn.reconnect()
			} else {
				content = nil
			}
		}
	}
}
func (conn *connector) read() {
	for {
		result, err := conn.socket.ReadString()
		if err != nil {
			conn.p.sendEvent(&Event{EventType: ErrorEvent, EventData: err})
			return
		}
		res, _ := jsonrpc.ParseString(result)
		if res.Type == jsonrpc.InvalidType {
			conn.p.sendEvent(&Event{EventType: ErrorEvent, EventData: err})
			continue
		}
		id, _ := res.ID.(string)
		rpc, ok := conn.rpcs.Get(id)
		if ok {
			if res.Error != nil {
				err = errors.New(res.Error.Message)
				rpc.Result <- &util.ResultItem{Result: nil, Err: err}
			} else {
				rpc.Result <- &util.ResultItem{Result: res.Result, Err: nil}
			}
			conn.rpcs.Delete(id)
		}
	}
}

// Request sending a jsonrpc Request object to a Server.
func (conn *connector) request(method string, args interface{}) (result interface{}, err error) {
	if !conn.isConnected() {
		return nil, errors.New("socket was closed")
	}
	requestid := conn.randID()
	item := &util.RequestItem{Expiretime: time.Now(), Result: make(chan *util.ResultItem)}
	conn.rpcs.Set(requestid, item)
	value, _ := jsonrpc.Request(requestid, method, args)
	conn.queue <- value
	timeout := time.NewTimer(conn.rpctimeout)
	select {
	case msg := <-item.Result:
		result = msg.Result
		err = msg.Err
	case <-timeout.C:
		err = errors.New("Send RPC time out," + requestid)
	}
	timeout.Stop()
	conn.rpcs.Delete(requestid)
	close(item.Result)
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
	conn.p.sendEvent(&Event{EventType: CloseEvent, EventData: nil})
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
			conn.p.sendEvent(&Event{EventType: ReconnectEvent, EventData: err})
			if err == nil {
				go conn.read()
				conn.setConnected(true)
				return
			}
		}
	}
}
func (conn *connector) createConn() (socket *service.Socket, err error) {
	tcpAddr, err := net.ResolveTCPAddr("tcp4", conn.addr)
	if err != nil {
		return nil, err
	}
	tcp, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		return nil, err
	}
	tcp.SetKeepAlive(true)
	socket = service.NewSocket(tcp)
	sign := conn.p.getSignature()
	if sign == "" {
		return socket, nil
	}
	reply, err := conn.auth(socket, sign)
	if err != nil {
		socket.Close()
		return nil, errors.New(reply)
	}
	err = conn.validateAuth(reply)
	return
}

// Auth for get authorization from server with sign
func (conn *connector) auth(socket *service.Socket, sign string) (reply string, err error) {
	_, err = socket.WriteBulkString([]byte(sign), time.Second*30)
	if err != nil {
		return
	}
	reply, err = socket.ReadString(time.Second * 30)
	return
}
func (conn *connector) validateAuth(reply string) error {
	res, _ := jsonrpc.ParseString(reply)
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
