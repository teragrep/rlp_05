package RelpConnection

import (
	"bytes"
	"crypto/tls"
	"errors"
	"github.com/teragrep/rlp_05/internal/Errors"
	"github.com/teragrep/rlp_05/internal/RelpCommand"
	"github.com/teragrep/rlp_05/internal/RelpParser"
	"github.com/teragrep/rlp_05/internal/RelpWindow"
	"github.com/teragrep/rlp_05/pkg/RelpBatch"
	"github.com/teragrep/rlp_05/pkg/RelpDialer"
	"github.com/teragrep/rlp_05/pkg/RelpFrame"
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
	RelpDialer.RelpDialer
	txId                 uint64
	rxBufferSize         int
	txBufferSize         int
	preAllocTxBuffer     *bytes.Buffer
	preAllocRxBuffer     []byte
	state                int
	Window               *RelpWindow.RelpWindow
	offer                []byte
	lastIp               string
	lastPort             int
	ackTimeoutDuration   time.Duration
	writeTimeoutDuration time.Duration
	TlsConfig            *tls.Config
}

// Init initializes the connection struct with CLOSED state and allocates the TX/RX buffers
func (relpConn *RelpConnection) Init() {
	relpConn.state = STATE_CLOSED
	relpConn.rxBufferSize = 512
	relpConn.txBufferSize = 262144
	relpConn.preAllocRxBuffer = make([]byte, relpConn.rxBufferSize)
	relpConn.preAllocTxBuffer = bytes.NewBuffer(make([]byte, 0, relpConn.txBufferSize))
	relpConn.txId = 0 // sendBatch() increments this by one before sending
	relpConn.Window = &RelpWindow.RelpWindow{}
	relpConn.offer = []byte("\nrelp_version=0\nrelp_software=RLP-05\ncommands=syslog\n")
	relpConn.ackTimeoutDuration = 30 * time.Second
	relpConn.writeTimeoutDuration = 30 * time.Second
	relpConn.TlsConfig = &tls.Config{}
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
	relpConn.Window.Init()

	encrypted, netErr := relpConn.RelpDialer.Dial(hostname, port, relpConn.TlsConfig)
	if netErr != nil {
		return false, &Errors.ConnectionEstablishmentError{
			Hostname:  hostname,
			Port:      port,
			Reason:    netErr.Error(),
			Encrypted: encrypted,
			Protocol:  "tcp",
		}
	}

	// send open session message
	relpRequest := RelpFrame.TX{
		Frame: RelpFrame.Frame{
			TransactionId: relpConn.txId,
			Cmd:           RelpCommand.RELP_OPEN,
			DataLength:    len(relpConn.offer),
			Data:          relpConn.offer,
		},
	}
	openerBatch := RelpBatch.RelpBatch{}
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
	relpRequest := RelpFrame.TX{Frame: RelpFrame.Frame{
		TransactionId: relpConn.txId,
		Cmd:           RelpCommand.RELP_CLOSE,
		DataLength:    0,
		Data:          nil,
	}}

	closerBatch := RelpBatch.RelpBatch{}
	closerBatch.Init()

	reqId := closerBatch.PutRequest(&relpRequest)
	err := relpConn.SendBatch(&closerBatch)
	success := false
	closeResp, err := closerBatch.GetResponse(reqId)
	if err == nil && closeResp != nil && closeResp.DataLength == 0 {
		success = true
	}

	if success {
		// if sending CLOSE command was successful, close connection and set state to CLOSED
		relpConn.TearDown()
	}

	return success
}

// Commit commits the RELP batch to the server
func (relpConn *RelpConnection) Commit(batch *RelpBatch.RelpBatch) error {
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
func (relpConn *RelpConnection) SendBatch(batch *RelpBatch.RelpBatch) error {
	log.Printf("SendBatch.Entry> Batch workQueue: %v request(s), Pending requests in window: %v\n",
		batch.GetWorkQueueLen(), len(relpConn.Window.Pending))
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
		relpRequest.TransactionId = relpConn.txId
		log.Printf("SendBatch> Sending request\n%v %v %v '%v'\nfrom batch\n", relpRequest.TransactionId, relpRequest.Cmd,
			relpRequest.DataLength, string(relpRequest.Data))

		relpConn.Window.PutPending(relpConn.txId, reqId)
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
func (relpConn *RelpConnection) ReadAcks(batch *RelpBatch.RelpBatch) error {
	log.Printf("ReadAcks.Entry> Reading ACKs for batchID: %v\n", batch.RequestId)
	var parser *RelpParser.RelpParser = nil
	notComplete := relpConn.Window.Size() > 0

	for notComplete { // until window is empty
		readBytes := 0
		for { // until parse complete
			if parser == nil {
				parser = &RelpParser.RelpParser{}
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
					return &Errors.AckReadingError{Reason: "timeout"}
				} else if err == io.EOF {
					return &Errors.AckReadingError{Reason: "eof"}
				} else {
					// other error
					return &Errors.AckReadingError{Reason: "unexpected error: " + err.Error()}
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

			if parser.IsComplete {
				log.Printf("ReadAcks> Parsing complete, with %v byte(s) read\n", readBytes)
				// resp read successfully
				txnId := parser.FrameTxnId
				if relpConn.Window.IsPending(txnId) {
					reqId, err := relpConn.Window.GetPending(txnId)
					if err != nil {
						panic("Could not find given pending txnId from RelpWindow!")
					}
					response := RelpFrame.RX{
						Frame: RelpFrame.Frame{
							TransactionId: parser.FrameTxnId,
							Cmd:           parser.FrameCmdString,
							DataLength:    parser.FrameLen,
							Data:          parser.FrameData.Bytes(),
						},
					}
					batch.PutResponse(reqId, &response)
					relpConn.Window.RemovePending(txnId)
				}

				parser = nil
				if relpConn.Window.Size() == 0 {
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
func (relpConn *RelpConnection) SendRelpRequest(tx *RelpFrame.TX) error {
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
