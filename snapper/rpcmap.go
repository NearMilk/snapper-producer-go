package snapper

import (
	"sync"
	"time"
)

type requestItem struct {
	expiretime time.Time
	callback   chan *resultItem
}
type resultItem struct {
	result interface{}
	err    error
}

func newrpcmap() *rpcmap {
	return &rpcmap{
		Mutex: &sync.Mutex{},
		rpcs:  make(map[string]*requestItem)}
}

type rpcmap struct {
	*sync.Mutex
	rpcs map[string]*requestItem
}

func (conn *rpcmap) delete(id string) {
	conn.Lock()
	defer conn.Unlock()
	delete(conn.rpcs, id)
}
func (conn *rpcmap) set(key string, val *requestItem) {
	conn.Lock()
	defer conn.Unlock()
	conn.rpcs[key] = val
}
func (conn *rpcmap) get(key string) (*requestItem, bool) {
	conn.Lock()
	defer conn.Unlock()
	rpc, ok := conn.rpcs[key]
	return rpc, ok
}
