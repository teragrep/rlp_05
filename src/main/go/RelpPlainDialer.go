package main

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"time"
)

// RelpPlainDialer contains the net.Conn struct used for unencrypted connections.
type RelpPlainDialer struct {
	connection *net.Conn
}

// Dial connects to the specified hostname and port
// Returns boolean if the connection is encrypted or not and possible errors as the second return value.
func (relpd *RelpPlainDialer) Dial(hostname string, port int, _ *tls.Config) (bool, error) {
	conn, err := net.Dial("tcp", fmt.Sprintf("%v:%v", hostname, port))
	if err != nil {
		return false, err
	} else {
		relpd.connection = &conn
	}
	return false, nil
}

// Write writes the byte array to the connection
func (relpd *RelpPlainDialer) Write(src []byte) (int, error) {
	if relpd.connection != nil {
		return (*relpd.connection).Write(src)
	}
	return -1, errors.New("plain connection not available for writing")
}

// Read reads the incoming data to the specified byte array
func (relpd *RelpPlainDialer) Read(dest []byte) (int, error) {
	if relpd.connection != nil {
		return (*relpd.connection).Read(dest)
	}
	return -1, errors.New("plain connection not available for reading")
}

// SetReadDeadline sets the deadline for reading. The given duration is added on current time.
func (relpd *RelpPlainDialer) SetReadDeadline(dur time.Duration) error {
	if relpd.connection != nil {
		return (*relpd.connection).SetReadDeadline(time.Now().Add(dur))
	}
	return errors.New("plain connection not available for read deadline configuration")
}

// SetWriteDeadline sets the deadline for writing. The given duration is added on current time.
func (relpd *RelpPlainDialer) SetWriteDeadline(dur time.Duration) error {
	if relpd.connection != nil {
		return (*relpd.connection).SetWriteDeadline(time.Now().Add(dur))
	}
	return errors.New("plain connection not available for write deadline configuration")
}

// Close closes the connection
func (relpd *RelpPlainDialer) Close() error {
	if relpd.connection != nil {
		return (*relpd.connection).Close()
	}
	return errors.New("plain connection not available to close the connection")
}
