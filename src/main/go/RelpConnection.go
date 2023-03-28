package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"time"
)

const MAX_COMMAND_LENGTH int = 11
const (
	STATE_CLOSED = 0
	STATE_OPEN   = 1
	STATE_COMMIT = 2
)

// RelpConnection struct contains the necessary fields to
// manage a TCP connection to the RELP server
type RelpConnection struct {
	txId             uint64
	rxBufferSize     int
	txBufferSize     int
	preAllocTxBuffer *bytes.Buffer
	preAllocRxBuffer *bytes.Buffer
	connection       *net.Conn
	state            int
	window           *RelpWindow
	offer            []byte
}

// Init initializes the connection struct with CLOSED state and allocates the TX/RX buffers
func (relpConn *RelpConnection) Init() {
	relpConn.state = STATE_CLOSED
	relpConn.rxBufferSize = 512
	relpConn.txBufferSize = 262144
	relpConn.preAllocRxBuffer = bytes.NewBuffer(make([]byte, 0, relpConn.rxBufferSize))
	relpConn.preAllocTxBuffer = bytes.NewBuffer(make([]byte, 0, relpConn.txBufferSize))
	relpConn.txId = 0 // sendBatch() increments this by one before sending
	relpConn.window = &RelpWindow{}
	relpConn.window.Init()
	relpConn.offer = []byte("\nrelp_version=0\nrelp_software=RLP-05\ncommands=syslog\n")
}

// Connect connects to the specified RELP server and sends OPEN message to initialize the connection.
// The returned boolean value specifies if the connection could be verified or not
func (relpConn *RelpConnection) Connect(hostname string, port int) bool {
	if relpConn.state != STATE_CLOSED {
		panic("Can't connect, the connection is not closed")
	}

	netConn, netErr := net.Dial("tcp", fmt.Sprintf("%v:%v", hostname, port))
	if netErr != nil {
		log.Fatal("RelpConnection: Could not dial TCP to address ", hostname, ":", port)
	} else {
		relpConn.connection = &netConn
	}

	// send open session message
	relpRequest := RelpFrameTX{
		RelpFrame{
			transactionId: relpConn.txId,
			cmd:           RELP_OPEN,
			dataLength:    len(relpConn.offer),
			data:          relpConn.offer,
		},
	}
	openerBatch := RelpBatch{}
	openerBatch.Init()

	reqId := openerBatch.PutRequest(&relpRequest)
	relpConn.SendBatch(&openerBatch)
	success := openerBatch.VerifyTransaction(reqId)
	if success {
		log.Println("[SUCCESS] Successfully opened connection to RELP server")
		relpConn.state = STATE_OPEN
	} else {
		log.Println("[FAIL] Connection failed, initial transaction could not be verified")
	}

	return success
}

// TearDown closes the connection to the server.
// The Disconnect method should be used instead.
func (relpConn *RelpConnection) TearDown() {
	var cn = *relpConn.connection
	err := cn.Close()
	if err != nil {
		log.Println("Error closing RELP connection")
	}
	relpConn.state = STATE_CLOSED
}

// Disconnect sends the CLOSE message to the server, and tries to disconnect gracefully.
// Calls the TearDown method if the CLOSE message was acknowledged by the server
func (relpConn *RelpConnection) Disconnect() bool {
	if relpConn.state != STATE_OPEN {
		panic("Cannot disconnect, connection was not OPEN")
	}
	relpRequest := RelpFrameTX{RelpFrame{
		transactionId: relpConn.txId,
		cmd:           RELP_CLOSE,
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
		relpConn.TearDown()
	}

	return success
}

// Commit commits the RELP batch to the server
func (relpConn *RelpConnection) Commit(batch *RelpBatch) {
	if relpConn.state != STATE_OPEN {
		panic("Can't commit, connection was in state other than OPEN.")
	}

	relpConn.state = STATE_COMMIT
	relpConn.SendBatch(batch)
	relpConn.state = STATE_OPEN
}

// SendBatch sends the RELP frames to the server in the given batch.
// The frames are sent asynchronously, and the server ACKs are checked after sending.
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

		// make sure txId loops 1 - 999 999 999
		if relpConn.txId >= 999_999_999 {
			relpConn.txId = 0
		}

		relpConn.txId += 1
		relpRequest.transactionId = relpConn.txId

		log.Println(relpRequest)
		relpConn.window.PutPending(relpConn.txId, reqId)

		go relpConn.SendRelpRequestAsync(relpRequest)
	}

	relpConn.ReadAcks(batch)
}

// ReadAcks reads the ACKs from the given batch.
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

			err2 := cn.SetReadDeadline(time.Now().Add(60 * time.Second))
			if err2 != nil {
				panic("Error setting timeout")
			}

			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					// reading timed out
					log.Fatalln("Reading ACK timed out (60 seconds)")
				} else if err == io.EOF {
					// EOF error
					log.Println("Encountered EOF in ACK")
					// write rest and break
					relpConn.preAllocRxBuffer.Write(tmp[0:n])
					readBytes += n
					break
				} else {
					// other error
					log.Fatalln(err)
				}
			}

			// write and break if line break encountered
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
	log.Println("ReadAcks: done")
}

// SendRelpRequestAsync sends the RELP frame to the connected RELP server
func (relpConn *RelpConnection) SendRelpRequestAsync(tx *RelpFrameTX) {
	var buf *bytes.Buffer
	if tx.dataLength > relpConn.txBufferSize {
		buf = bytes.NewBuffer(make([]byte, 0, tx.dataLength))
		relpConn.preAllocTxBuffer = buf
		relpConn.txBufferSize = buf.Cap()
	} else {
		buf = relpConn.preAllocTxBuffer
	}

	tx.Write(buf)
	var cn = *relpConn.connection
	n, err := cn.Write(buf.Bytes())
	if err != nil {
		log.Fatalln("Could not write bytes to server")
	} else {
		log.Println(n, "bytes written to server")
	}

	buf.Reset()
}
