package snapper

import (
	"bufio"
	"errors"
	"net"
	"time"

	"github.com/teambition/respgo"
)

// newSocket return new socket
func newSocket(conn net.Conn) *socket {
	return &socket{Conn: conn, reader: bufio.NewReader(conn)}
}

// socket ...
type socket struct {
	net.Conn
	reader *bufio.Reader
}

// ReadLine return treated message according to RESP Protocol
func (s *socket) ReadLine(timeouts ...time.Duration) (result interface{}, err error) {
	if len(timeouts) > 0 {
		s.SetReadDeadline(time.Now().Add(timeouts[0]))
		defer s.SetReadDeadline(time.Time{})
	}
	return respgo.Decode(s.reader)
}

// ReadString return treated SimpleStrings or BulkStrings message according to RESP Protocol
func (s *socket) ReadString(timeouts ...time.Duration) (result string, err error) {
	val, err := s.ReadLine(timeouts...)
	if err != nil {
		return
	}
	var ok bool
	if result, ok = val.(string); !ok {
		err = errors.New("invalid string or bulkstring type")
	}
	return
}

// Write writes data to the connection.
// Write can with timeout arguments and return an Error with Timeout() == true after a fixed time limit.
func (s *socket) Write(b []byte, timeouts ...time.Duration) (int, error) {
	if len(timeouts) > 0 {
		s.SetWriteDeadline(time.Now().Add(timeouts[0]))
		defer s.SetWriteDeadline(time.Time{})
	}
	return s.Conn.Write(b)
}

// WriteBulkString writes resp bulkstring to the connection.
// WriteBulkString can with timeout arguments and return an Error with Timeout() == true after a fixed time limit.
func (s *socket) WriteBulkString(str string, timeouts ...time.Duration) (int, error) {
	bytes := respgo.EncodeBulkString(str)
	return s.Write(bytes, timeouts...)
}
