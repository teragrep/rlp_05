package main

// RelpFrame is the base struct for response and request frame structs
type RelpFrame struct {
	transactionId uint64
	cmd           string
	dataLength    int
	data          []byte
}
