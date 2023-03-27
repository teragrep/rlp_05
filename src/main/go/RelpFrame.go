package main

type RelpFrame struct {
	transactionId uint64
	cmd           string
	dataLength    int
	data          []byte
}

// TODO: readString()?
