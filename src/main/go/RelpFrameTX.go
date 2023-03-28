package main

import (
	"bytes"
	"log"
	"strconv"
)

// RelpFrameTX is a struct containing a request RELP frame
type RelpFrameTX struct {
	RelpFrame
}

const (
	SP byte = 32 // byte for ' '
	NL byte = 10 // byte for \n
)

// Write writes the frame to a relp message in the buffer
func (txFrame *RelpFrameTX) Write(byteBuf *bytes.Buffer) *bytes.Buffer {
	log.Println("RelpFrameTX: Start writing to buffer...")
	// transaction id
	byteBuf.Write([]byte(strconv.FormatUint(txFrame.transactionId, 10)))
	// command
	byteBuf.WriteByte(SP)
	byteBuf.Write([]byte(txFrame.cmd))
	// data length
	byteBuf.WriteByte(SP)
	byteBuf.Write([]byte(strconv.FormatUint(uint64(txFrame.dataLength), 10)))
	// data
	byteBuf.WriteByte(SP)
	byteBuf.Write(txFrame.data)
	byteBuf.WriteByte(NL)

	log.Printf("RelpFrameTX: Wrote %v byte(s) to buffer, string:\n%v\n", byteBuf.Len(), string(byteBuf.Bytes()))

	return byteBuf
}
