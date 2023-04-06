package RelpDialer

import (
	"crypto/tls"
	"errors"
	"fmt"
	"time"
)

// RelpTLSDialer contains the encrypted tls.Conn connection struct
type RelpTLSDialer struct {
	connection *tls.Conn
}

// Dial sets up the encrypted connection using the given tls.Config
// Returns boolean if the connection is encrypted or not and possible errors as the second return value.
func (relpd *RelpTLSDialer) Dial(hostname string, port int, cfg *tls.Config) (bool, error) {
	conn, err := tls.Dial("tcp", fmt.Sprintf("%v:%v", hostname, port), cfg)
	if err != nil {
		return true, err
	} else {
		relpd.connection = conn
	}
	return true, nil
}

// Write writes the given byte array to the connection
func (relpd *RelpTLSDialer) Write(src []byte) (int, error) {
	if relpd.connection != nil {
		return relpd.connection.Write(src)
	}
	return -1, errors.New("tls connection not available for writing")
}

// Read reads the data from the connection to the given byte array
func (relpd *RelpTLSDialer) Read(dest []byte) (int, error) {
	if relpd.connection != nil {
		return relpd.connection.Read(dest)
	}
	return -1, errors.New("tls connection not available for reading")
}

// SetReadDeadline sets the deadline for reading operations. The duration is added on current time.
func (relpd *RelpTLSDialer) SetReadDeadline(dur time.Duration) error {
	if relpd.connection != nil {
		return relpd.connection.SetReadDeadline(time.Now().Add(dur))
	}
	return errors.New("tls connection not available for read deadline configuration")
}

// SetWriteDeadline sets the deadline for writing operations. The duration is added on current time.
func (relpd *RelpTLSDialer) SetWriteDeadline(dur time.Duration) error {
	if relpd.connection != nil {
		return relpd.connection.SetWriteDeadline(time.Now().Add(dur))
	}
	return errors.New("tls connection not available for write deadline configuration")
}

// Close closes the connection.
func (relpd *RelpTLSDialer) Close() error {
	if relpd.connection != nil {
		return relpd.connection.Close()
	}
	return errors.New("tls connection not available to close the connection")
}
