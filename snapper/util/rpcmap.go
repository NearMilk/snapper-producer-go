package util

import (
	"sync"
	"time"
)

// RequestItem ...
type RequestItem struct {
	Expiretime time.Time
	Result     chan *ResultItem
}

// ResultItem ...
type ResultItem struct {
	Result interface{}
	Err    error
}

// NewRPCMap ...
func NewRPCMap() *RPCMap {
	return &RPCMap{
		Mutex: &sync.Mutex{},
		rpcs:  make(map[string]*RequestItem)}
}

// RPCMap ...
type RPCMap struct {
	*sync.Mutex
	rpcs map[string]*RequestItem
}

// Delete ...
func (conn *RPCMap) Delete(id string) {
	conn.Lock()
	defer conn.Unlock()
	delete(conn.rpcs, id)
}

// Set ...
func (conn *RPCMap) Set(key string, val *RequestItem) {
	conn.Lock()
	defer conn.Unlock()
	conn.rpcs[key] = val
}

// Get ...
func (conn *RPCMap) Get(key string) (*RequestItem, bool) {
	conn.Lock()
	defer conn.Unlock()
	rpc, ok := conn.rpcs[key]
	return rpc, ok
}
