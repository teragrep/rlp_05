package main

import (
	"crypto/tls"
	"time"
)

// RelpDialer is the interface for all the methods RELP dialers
// should contain. Check RelpPlainDialer and RelpTLSDialer for implementations.
type RelpDialer interface {
	// Dial dials to the given hostname and port, providing the tls.Config if necessary. Returns if the connection is encrypted, and
	// if any errors were encountered.
	Dial(hostname string, port int, cfg *tls.Config) (bool, error)
	// SetReadDeadline sets the deadline for reading operations. The duration is added on top of current time.
	SetReadDeadline(add time.Duration) error
	// SetWriteDeadline sets the deadline for writing operations. The duration is added on top of current time.
	SetWriteDeadline(add time.Duration) error
	// Write writes the src byte array to the connection. Returns the written amount of bytes and any possible errors.
	Write(src []byte) (int, error)
	// Read reads the data from the connection to the dest byte array. Returns the amount of bytes read and any possible errors.
	Read(dest []byte) (int, error)
	// Close closes the connection, returning any possible errors.
	Close() error
}
