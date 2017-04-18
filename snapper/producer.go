package snapper

import (
	"errors"
	"time"

	"github.com/SermoDigital/jose/crypto"
	"github.com/SermoDigital/jose/jws"
	"github.com/teambition/jsonrpc-go"
)

// Producer form send request and message to remote server
type Producer struct {
	queue [][]string
	conn  *connector
	opts  *Options
	Event chan *Event
}

// Options for Producer
type Options struct {
	ProducerID       string   // producer's name, use for log.
	SecretKeys       []string // A array of string or buffer containing the secret for HMAC algorithms.
	ExpiresIn        int64    // jwt expire time, default to 3600 * 24, one day. Note:this must be in seconds.
	Address          string   // defaut to "127.0.0.1:7720"
	MessageWaitCount int      // default to 10000.
	Rpctimeout       int      // default to 40 seconds

}

//New an Producer
func New(opts *Options) (producer *Producer, err error) {
	initOpts(opts)
	// producer
	producer = &Producer{queue: make([][]string, 1)}
	producer.opts = opts
	producer.Event = make(chan *Event, 100)
	// connector
	conn := newConnector(opts.Address, producer, time.Duration(opts.Rpctimeout)*time.Second)
	producer.conn = conn
	err = producer.conn.start()
	return
}

// SignAuth Generate a token string. payload should have userId property and type property
// that Snapper server determine which room the consumer should auto join.
func (p *Producer) SignAuth(arg map[string]interface{}) (string, error) {
	claims := jws.Claims{}
	claims.SetExpiration(time.Now().Add(time.Duration(p.opts.ExpiresIn) * time.Second))
	for key, val := range arg {
		claims.Set(key, val)
	}
	jwt := jws.NewJWT(claims, crypto.SigningMethodHS256)
	b, err := jwt.Serialize([]byte(p.opts.SecretKeys[0]))
	return string(b), err
}

// SendMessage message to specific room
func (p *Producer) SendMessage(room string, msg string) (err error) {
	if p.conn.notificationQueue.Len() > p.opts.MessageWaitCount {
		err = errors.New("the message wait count is already maxed out")
	} else {
		p.conn.notification([]string{room, msg})
	}
	return
}

// JoinRoom with specific room and consumerId
func (p *Producer) JoinRoom(room string, consumerID string) (result interface{}, err error) {
	return p.Request("subscribe", []string{room, consumerID})
}

// LeaveRoom  with specific room and consumerId
func (p *Producer) LeaveRoom(room string, consumerID string) (result interface{}, err error) {
	return p.Request("unsubscribe", []string{room, consumerID})
}

// Request sending a jsonrpc Request Call to a Server.
func (p *Producer) Request(method string, args interface{}) (result interface{}, err error) {
	return p.conn.request(method, args)
}

// Close the current Producer
func (p *Producer) Close() {
	p.conn.close()
}

func initOpts(opts *Options) {
	if opts.ExpiresIn < 1 {
		opts.ExpiresIn = 86400 // 3600 * 24
	}
	if opts.Address == "" {
		opts.Address = "127.0.0.1:7720"
	}
	if opts.MessageWaitCount < 1 {
		opts.MessageWaitCount = 10000
	}
	if opts.Rpctimeout < 1 {
		opts.Rpctimeout = 40
	}
}
func (p *Producer) getSignature() string {
	token, _ := p.SignAuth(map[string]interface{}{"id": p.opts.ProducerID})
	buf, _ := jsonrpc.Request(p.conn.randID(), "auth", []string{token})
	return string(buf)
}
func (p *Producer) sendEvent(event *Event) {
	p.Event <- event
}
