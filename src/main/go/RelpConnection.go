package main

import (
	"bytes"
	"crypto/tls"
	"errors"
	"io"
	"log"
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
	RelpDialer
	txId                 uint64
	rxBufferSize         int
	txBufferSize         int
	preAllocTxBuffer     *bytes.Buffer
	preAllocRxBuffer     []byte
	state                int
	window               *RelpWindow
	offer                []byte
	lastIp               string
	lastPort             int
	ackTimeoutDuration   time.Duration
	writeTimeoutDuration time.Duration
	tlsConfig            *tls.Config
}

// Init initializes the connection struct with CLOSED state and allocates the TX/RX buffers
func (relpConn *RelpConnection) Init() {
	relpConn.state = STATE_CLOSED
	relpConn.rxBufferSize = 512
	relpConn.txBufferSize = 262144
	relpConn.preAllocRxBuffer = make([]byte, relpConn.rxBufferSize)
	relpConn.preAllocTxBuffer = bytes.NewBuffer(make([]byte, 0, relpConn.txBufferSize))
	relpConn.txId = 0 // sendBatch() increments this by one before sending
	relpConn.window = &RelpWindow{}
	relpConn.offer = []byte("\nrelp_version=0\nrelp_software=RLP-05\ncommands=syslog\n")
	relpConn.ackTimeoutDuration = 30 * time.Second
	relpConn.writeTimeoutDuration = 30 * time.Second
	relpConn.tlsConfig = &tls.Config{}
}

// Connect connects to the specified RELP server and sends OPEN message to initialize the connection.
// The returned boolean value specifies if the connection could be verified or not
func (relpConn *RelpConnection) Connect(hostname string, port int) (bool, error) {
	if relpConn.state != STATE_CLOSED {
		panic("Can't connect, the connection is not closed")
	}

	if relpConn.RelpDialer == nil {
		panic("Can't connect, RelpDialer has not been set! Please set as RelpTLSDialer or RelpPlainDialer." +
			" Use the relpConnection.tlsConfig to configure the TLS connection!")
	}

	// save used IP and port in case of needing to reconnect
	relpConn.lastIp = hostname
	relpConn.lastPort = port

	// reset txId & relpWindow
	relpConn.txId = 0
	relpConn.window.Init()

	encrypted, netErr := relpConn.RelpDialer.Dial(hostname, port, relpConn.tlsConfig)
	if netErr != nil {
		return false, &ConnectionEstablishmentError{
			hostname:  hostname,
			port:      port,
			reason:    netErr.Error(),
			encrypted: encrypted,
			protocol:  "tcp",
		}
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
	err := relpConn.SendBatch(&openerBatch)
	success := openerBatch.VerifyTransaction(reqId)
	if success {
		log.Println("[SUCCESS] Successfully opened connection to RELP server")
		relpConn.state = STATE_OPEN
	} else {
		log.Println("[FAIL] Connection failed, initial transaction could not be verified")
	}

	return success, err
}

// TearDown closes the connection to the server.
// The Disconnect method should be used instead.
func (relpConn *RelpConnection) TearDown() {
	err := relpConn.RelpDialer.Close()
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
	err := relpConn.SendBatch(&closerBatch)
	success := false
	closeResp, err := closerBatch.GetResponse(reqId)
	if err == nil && closeResp != nil && closeResp.dataLength == 0 {
		success = true
	}

	if success {
		// if sending CLOSE command was successful, close connection and set state to CLOSED
		relpConn.TearDown()
	}

	return success
}

// Commit commits the RELP batch to the server
func (relpConn *RelpConnection) Commit(batch *RelpBatch) error {
	if relpConn.state != STATE_OPEN {
		panic("Can't commit, connection was in state other than OPEN.")
	}

	relpConn.state = STATE_COMMIT
	err := relpConn.SendBatch(batch)
	relpConn.state = STATE_OPEN
	return err
}

// SendBatch sends the RELP frames to the server in the given batch.
// The frames are sent asynchronously, and the server ACKs are checked after sending.
func (relpConn *RelpConnection) SendBatch(batch *RelpBatch) error {
	log.Printf("SendBatch.Entry> Batch workQueue: %v request(s), Pending requests in window: %v\n",
		batch.GetWorkQueueLen(), len(relpConn.window.pending))
	// send a batch of requests
	for batch.GetWorkQueueLen() > 0 {
		reqId := batch.PopWorkQueue()
		relpRequest, err := batch.GetRequest(reqId)
		if err != nil {
			log.Fatalln("Could not get request from batch")
		}

		// relp Request-Response txId
		// <txId is here> <command> <len> <data> NL
		// make sure txId loops 1 - 999 999 999
		if relpConn.txId >= 999_999_999 {
			relpConn.txId = 1
		} else {
			relpConn.txId++
		}
		relpRequest.transactionId = relpConn.txId
		log.Printf("SendBatch> Sending request\n%v %v %v '%v'\nfrom batch\n", relpRequest.transactionId, relpRequest.cmd,
			relpRequest.dataLength, string(relpRequest.data))

		relpConn.window.PutPending(relpConn.txId, reqId)
		log.Println("SendBatch> Put pending: ", relpConn.txId, reqId)

		sendErr := relpConn.SendRelpRequest(relpRequest)
		if sendErr != nil {
			log.Printf("Error sending relp request: '%v'\n", err.Error())
		}

		ackErr := relpConn.ReadAcks(batch)
		if ackErr != nil {
			// ACK timeout or other failure
			return ackErr
		}
	}

	return nil
}

// ReadAcks reads the ACKs from the given batch.
func (relpConn *RelpConnection) ReadAcks(batch *RelpBatch) error {
	log.Printf("ReadAcks.Entry> Reading ACKs for batchID: %v\n", batch.requestId)
	var parser *RelpParser = nil
	notComplete := relpConn.window.Size() > 0

	for notComplete { // until window is empty
		readBytes := 0
		for { // until parse complete
			if parser == nil {
				parser = &RelpParser{}
			}

			// set ACK timeout duration, default 30 sec
			errDl := relpConn.RelpDialer.SetReadDeadline(relpConn.ackTimeoutDuration)
			if errDl != nil {
				return errors.New("error setting connection timeout")
			}
			n, err := relpConn.RelpDialer.Read(relpConn.preAllocRxBuffer)

			if err != nil {
				if errors.Is(err, os.ErrDeadlineExceeded) {
					// reading timed out
					return &AckReadingError{reason: "timeout"}
				} else if err == io.EOF {
					return &AckReadingError{reason: "eof"}
				} else {
					// other error
					return &AckReadingError{reason: "unexpected error: " + err.Error()}
				}
			} else {
				readBytes += n
			}

			// parse all bytes in buffer
			for i := 0; i < n; i++ {
				parseErr := parser.Parse(relpConn.preAllocRxBuffer[i])
				if parseErr != nil {
					panic("parsing error: " + parseErr.Error())
				}
			}

			if parser.isComplete {
				log.Printf("ReadAcks> Parsing complete, with %v byte(s) read\n", readBytes)
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
					// window empty, can exit readAcks method
					notComplete = false
				}
				break
			}
		}

	}
	log.Println("ReadAcks.Done> Return with no errors")
	return nil
}

// SendRelpRequest sends the RELP frame to the connected RELP server
func (relpConn *RelpConnection) SendRelpRequest(tx *RelpFrameTX) error {
	txN, err := tx.Write(relpConn.preAllocTxBuffer)

	if err != nil {
		return err
	}

	dlErr := relpConn.RelpDialer.SetWriteDeadline(relpConn.writeTimeoutDuration)
	if dlErr != nil {
		return dlErr
	}
	n, writeErr := relpConn.RelpDialer.Write(relpConn.preAllocTxBuffer.Bytes())

	if writeErr != nil {
		return writeErr
	} else {
		log.Printf("SendRelpRequest> Total of %v byte(s) written to server from %v given byte(s). (%v%%)",
			n, txN, (1.00*n/txN)*100.0)
	}

	relpConn.preAllocTxBuffer.Reset()
	return nil
}
