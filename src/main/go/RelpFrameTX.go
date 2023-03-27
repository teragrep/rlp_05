package main

import (
	"bytes"
	"fmt"
	"log"
	"strconv"
)

type RelpFrameTX struct {
	RelpFrame
}

const SP byte = 32
const NL byte = 10

func (txFrame *RelpFrameTX) Write(byteBuf *bytes.Buffer) *bytes.Buffer {
	log.Println("RelpFrameTX: Write to buffer")
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

	//log.Printf("Wrote: %v to buffer", byteBuf.Bytes())
	//log.Printf("RelpFrameTX.Write: %v", string(byteBuf.Bytes()))
	fmt.Println(string(byteBuf.Bytes()))

	return byteBuf
}
