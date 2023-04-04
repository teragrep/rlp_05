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
func (txFrame *RelpFrameTX) Write(byteBuf *bytes.Buffer) (int, error) {
	idBytes := []byte(strconv.FormatUint(txFrame.transactionId, 10))
	cmdBytes := []byte(txFrame.cmd)
	dataLenBytes := []byte(strconv.FormatUint(uint64(txFrame.dataLength), 10))
	bytesWritten := 0

	log.Println("RelpFrameTX: Start writing to buffer...")
	// transaction id
	nId, errId := byteBuf.Write(idBytes)
	if errId != nil {
		return bytesWritten, errId
	} else {
		bytesWritten += nId
	}

	errSp1 := byteBuf.WriteByte(SP)
	if errSp1 != nil {
		return bytesWritten, errSp1
	} else {
		bytesWritten += 1
	}

	// command
	nCmd, errCmd := byteBuf.Write(cmdBytes)
	if errCmd != nil {
		return bytesWritten, errCmd
	} else {
		bytesWritten += nCmd
	}

	errSp2 := byteBuf.WriteByte(SP)
	if errSp2 != nil {
		return bytesWritten, errSp2
	} else {
		bytesWritten += 1
	}

	// data length
	nLen, errLen := byteBuf.Write(dataLenBytes)
	if errLen != nil {
		return bytesWritten, errLen
	} else {
		bytesWritten += nLen
	}

	errSp3 := byteBuf.WriteByte(SP)
	if errSp3 != nil {
		return bytesWritten, errSp3
	} else {
		bytesWritten += 1
	}

	// data
	nData, errData := byteBuf.Write(txFrame.data)
	if errData != nil {
		return bytesWritten, errData
	} else {
		bytesWritten += nData
	}

	errNl := byteBuf.WriteByte(NL)
	if errNl != nil {
		return bytesWritten, errNl
	} else {
		bytesWritten += 1
	}

	log.Printf("RelpFrameTX: Wrote %v byte(s) to buffer\n", bytesWritten)

	return bytesWritten, nil
}
