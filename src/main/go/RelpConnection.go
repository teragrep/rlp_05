package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net"
)

const MAX_COMMAND_LENGTH int = 11
const OFFER string = "\nrelp_version=0\nrelp_software=RLP-05\ncommands=syslog\n"
const (
	STATE_CLOSED = 0
	STATE_OPEN   = 1
	STATE_COMMIT = 2
)

type RelpConnection struct {
	txId             uint64
	rxBufferSize     int
	txBufferSize     int
	preAllocTxBuffer *bytes.Buffer
	preAllocRxBuffer *bytes.Buffer
	connection       *net.Conn
	state            int
	window           *RelpWindow
}

func (relpConn *RelpConnection) Init() {
	relpConn.state = STATE_CLOSED
	relpConn.rxBufferSize = 512
	relpConn.txBufferSize = 262144
	relpConn.preAllocRxBuffer = bytes.NewBuffer(make([]byte, 0, 512))
	relpConn.preAllocTxBuffer = bytes.NewBuffer(make([]byte, 0, 262144))
}

func (relpConn *RelpConnection) Connect(hostname string, port int) bool {
	if relpConn.state != STATE_CLOSED {
		panic("Can't connect, the connection is not closed")
	}

	relpConn.txId = 0 // FIXME: 1 - 999,999,999 with loopback at end;; sendBatch() increments this by one before sending
	relpConn.window = &RelpWindow{}
	relpConn.window.Init()
	netConn, netErr := net.Dial("tcp", fmt.Sprintf("%v:%v", hostname, port))
	if netErr != nil {
		log.Fatal("Could not dial TCP at address ", hostname, port)
	} else {
		relpConn.connection = &netConn
	}

	// send open session message
	relpRequest := RelpFrameTX{
		RelpFrame{
			transactionId: relpConn.txId,
			cmd:           "open",
			dataLength:    len([]byte(OFFER)),
			data:          []byte(OFFER),
		},
	}
	openerBatch := RelpBatch{}
	openerBatch.Init()

	reqId := openerBatch.PutRequest(&relpRequest)
	relpConn.SendBatch(&openerBatch)
	success := openerBatch.VerifyTransaction(reqId)
	if success {
		log.Println("Successfully opened connection")
		relpConn.state = STATE_OPEN
	} else {
		log.Println("Connection failed, could not be verified")
	}

	return success
}

func (relpConn *RelpConnection) TearDown() {
	var cn net.Conn = *relpConn.connection
	err := cn.Close()
	if err != nil {
		log.Println("Error closing relp connection")
	}
	relpConn.state = STATE_CLOSED
}

func (relpConn *RelpConnection) Disconnect() bool {
	if relpConn.state != STATE_OPEN {
		panic("Cannot disconnect, connection was not open")
	}
	relpRequest := RelpFrameTX{RelpFrame{
		transactionId: relpConn.txId,
		cmd:           "close",
		dataLength:    0,
		data:          nil,
	}}

	closerBatch := RelpBatch{}
	closerBatch.Init()

	reqId := closerBatch.PutRequest(&relpRequest)
	relpConn.SendBatch(&closerBatch)
	success := false
	closeResp, err := closerBatch.GetResponse(reqId)
	if err == nil && closeResp != nil && closeResp.dataLength == 0 {
		success = true
	}

	if success {
		var cn net.Conn = *relpConn.connection
		err := cn.Close()
		if err != nil {
			log.Println("Could not close connection in Disconnect method")
		}
		relpConn.state = STATE_CLOSED
	}

	return success
}

func (relpConn *RelpConnection) Commit(batch *RelpBatch) {
	if relpConn.state != STATE_OPEN {
		panic("Can't commit, connection was in state other than OPEN.")
	}

	relpConn.state = STATE_COMMIT
	relpConn.SendBatch(batch)
	relpConn.state = STATE_OPEN
}

func (relpConn *RelpConnection) SendBatch(batch *RelpBatch) {
	// send a batch of requests

	for batch.GetWorkQueueLen() > 0 {
		reqId := batch.PopWorkQueue()
		relpRequest, err := batch.GetRequest(reqId)
		if err != nil {
			log.Fatalln("Could not get request from batch")
		} else {
			log.Printf("Sending request %v from batch\n", relpRequest)
		}
		relpConn.txId += 1
		relpRequest.transactionId = relpConn.txId

		log.Println(relpRequest)
		relpConn.window.PutPending(relpConn.txId, reqId)

		relpConn.SendRelpRequestAsync(relpRequest)
	}

	relpConn.ReadAcks(batch)
}

func (relpConn *RelpConnection) ReadAcks(batch *RelpBatch) {
	log.Printf("Reading ACKs for batchID: %v\n", batch.requestId)
	var parser *RelpParser = nil
	notComplete := relpConn.window.Size() > 0
	var cn = *relpConn.connection

	for notComplete {
		tmp := make([]byte, 64)
		readBytes := 0
		for {
			n, err := cn.Read(tmp)

			if err != nil {
				if err != io.EOF {
					log.Fatalln("Could not read ack from batch")
				} else {
					log.Println("Encountered EOF in ACK")
					break
				}
			}

			relpConn.preAllocRxBuffer.Write(tmp[0:n])
			readBytes += n
			if tmp[n-1] == '\n' {
				break
			}
		}

		parsedBytes := 0
		if readBytes > 0 {
			log.Printf("Read %v byte(s) as ACK\n", readBytes)
			for parsedBytes < readBytes {
				if parser == nil {
					parser = &RelpParser{}
				}

				nextBytes := relpConn.preAllocRxBuffer.Next(1) // len always 1
				//log.Printf("Parsing byte: %v (str: %v)", nextBytes[0], string(nextBytes[0]))
				parser.Parse(nextBytes[0])
				parsedBytes++

				if parser.isComplete {
					log.Printf("ReadAcks: Parsing complete\n")
					// resp read successfully
					txnId := parser.frameTxnId
					if relpConn.window.IsPending(txnId) {
						reqId, err := relpConn.window.GetPending(txnId)
						if err != nil {
							panic("Could not find given pending txnId from RelpWindow!")
						}
						response := RelpFrameRX{
							RelpFrame{
								transactionId: parser.frameTxnId,
								cmd:           parser.frameCmdString,
								dataLength:    parser.frameLen,
								data:          parser.frameData.Bytes(),
							},
						}
						batch.PutResponse(reqId, &response)
						relpConn.window.RemovePending(txnId)
					}

					parser = nil
					if relpConn.window.Size() == 0 {
						notComplete = false
						break
					}
				}

			}
		}
		// everything is read now
		relpConn.preAllocRxBuffer.Reset()
	}
	log.Println("ReadAcks: exit")
}

func (relpConn *RelpConnection) SendRelpRequestAsync(tx *RelpFrameTX) {
	var buf *bytes.Buffer = bytes.NewBuffer(make([]byte, 0, tx.dataLength))
	// FIXME? server does not seem to like sending overlarge buffers
	if tx.dataLength > relpConn.txBufferSize {
		buf = bytes.NewBuffer(make([]byte, 0, tx.dataLength))
		relpConn.preAllocTxBuffer = buf
	} else {
		buf = relpConn.preAllocTxBuffer
	}

	tx.Write(buf)
	var cn = *relpConn.connection
	n, err := cn.Write(buf.Bytes())
	if err != nil {
		log.Fatalln("Could not write bytes to net.Conn")
	} else {
		log.Println(n, "bytes written to net.Conn")
	}

	buf.Reset()
}
