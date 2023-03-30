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
func (txFrame *RelpFrameTX) Write() ([]byte, int) {
	byteBuf := bytes.NewBuffer(make([]byte, 0, txFrame.dataLength+64))
	bytesWritten := 0
	log.Println("RelpFrameTX: Start writing to buffer...")
	// transaction id
	idBytes := []byte(strconv.FormatUint(txFrame.transactionId, 10))
	byteBuf.Write(idBytes)
	// command
	byteBuf.WriteByte(SP)
	cmdBytes := []byte(txFrame.cmd)
	byteBuf.Write(cmdBytes)
	// data length
	byteBuf.WriteByte(SP)
	dataLenBytes := []byte(strconv.FormatUint(uint64(txFrame.dataLength), 10))
	byteBuf.Write(dataLenBytes)
	// data
	byteBuf.WriteByte(SP)
	byteBuf.Write(txFrame.data)
	byteBuf.WriteByte(NL)

	bytesWritten = len(idBytes) + len(cmdBytes) + len(dataLenBytes) + txFrame.dataLength + 4
	log.Printf("RelpFrameTX: Wrote %v byte(s) to buffer\n", bytesWritten)

	return byteBuf.Bytes(), bytesWritten
}
