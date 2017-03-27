package snapper

import "sync"

// NewQueue ...
func NewQueue() *Queue {
	q := &Queue{sync: &sync.Mutex{}}
	return q
}

// Queue ...
type Queue struct {
	sync  *sync.Mutex
	array []interface{}
}

// Push ...
func (q *Queue) Push(arg interface{}) {
	q.sync.Lock()
	defer q.sync.Unlock()
	q.array = append(q.array, arg)
}

// Pop ...
func (q *Queue) Pop() (arr interface{}) {
	q.sync.Lock()
	defer q.sync.Unlock()
	if len(q.array) > 0 {
		arr = q.array[0]
		q.array = q.array[1:]
	}
	return arr
}

// PopN ...
func (q *Queue) PopN(num int) []interface{} {
	q.sync.Lock()
	defer q.sync.Unlock()
	length := len(q.array)
	if length < num {
		num = length
	}
	arr := make([]interface{}, num)
	copy(arr, q.array[:num])
	q.array = q.array[num:]
	return arr
}

// Len ...
func (q *Queue) Len() int {
	q.sync.Lock()
	defer q.sync.Unlock()
	return len(q.array)
}
