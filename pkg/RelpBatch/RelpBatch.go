package RelpBatch

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/teragrep/rlp_05/internal/RelpCommand"
	"github.com/teragrep/rlp_05/pkg/RelpFrame"
	"log"
)

// RelpBatch struct contains all the request frames and their response counterparts.
// the workQueue is used to keep track of the current, yet-to-be processed requests.
type RelpBatch struct {
	requests  map[uint64]*RelpFrame.TX
	responses map[uint64]*RelpFrame.RX
	workQueue *list.List
	RequestId uint64
}

// Init initializes the batch with new maps and list
func (batch *RelpBatch) Init() {
	batch.requests = make(map[uint64]*RelpFrame.TX)
	batch.responses = make(map[uint64]*RelpFrame.RX)
	batch.workQueue = list.New()
	batch.RequestId = 0 // id within this batch
}

// Insert inserts the given byte array syslog message;
// id SP syslog SP dataLength SP data NL
// Works similarly to calling PutRequest with a syslog message request frame
func (batch *RelpBatch) Insert(syslogMsg []byte) uint64 {
	relpRequest := RelpFrame.TX{
		Frame: RelpFrame.Frame{
			Data:       syslogMsg,
			DataLength: len(syslogMsg),
			Cmd:        RelpCommand.RELP_SYSLOG,
		},
	}

	return batch.PutRequest(&relpRequest)
}

// PutRequest puts the given request frame to the requests map and work queue
// batch.requestId is different from tx.transactionId
// !!! requestId resets each batch but transactionId is the same for all for one relp session
func (batch *RelpBatch) PutRequest(tx *RelpFrame.TX) uint64 {
	batch.RequestId += 1
	batch.requests[batch.RequestId] = tx
	batch.workQueue.PushBack(batch.RequestId)

	return batch.RequestId
}

// GetRequest gets the request frame from the requests map, if found.
// Otherwise will send a "could not find batch <id> request" error
func (batch *RelpBatch) GetRequest(id uint64) (*RelpFrame.TX, error) {
	v, ok := batch.requests[id]
	if ok {
		return v, nil
	} else {
		return nil, errors.New(fmt.Sprintf("could not find batch %v request", id))
	}
}

// RemoveRequest removes the specified request from the map and work queue
func (batch *RelpBatch) RemoveRequest(id uint64) {
	// remove from requests map
	delete(batch.requests, id)

	// find element to remove, and remove it using List.Remove
	elem := batch.workQueue.Front()
	for elem != nil {
		if elem.Value == id {
			batch.workQueue.Remove(elem)
			break
		}
		elem = elem.Next()
	}
}

// GetResponse gets the specified request from the map, if found
// Otherwise, returns "could not find batch <id> response" error
func (batch *RelpBatch) GetResponse(id uint64) (*RelpFrame.RX, error) {
	v, ok := batch.responses[id]
	if ok {
		return v, nil
	} else {
		return nil, errors.New(fmt.Sprintf("could not find batch %v response", id))
	}
}

// PutResponse puts the specified response frame to the response map
func (batch *RelpBatch) PutResponse(id uint64, response *RelpFrame.RX) {
	_, ok := batch.requests[id]
	if ok {
		batch.responses[id] = response
	}
}

// VerifyTransaction verifies, that the id given has a matching request and response frame saved,
// and that the response code is 200 OK
func (batch *RelpBatch) VerifyTransaction(id uint64) bool {
	log.Printf("Verifying transaction (batch-specific id, NOT txnId): %v\n", id)
	req, hasRequest := batch.requests[id]
	if hasRequest {
		log.Printf("Verify: Got request: %v %v %v %v\n", req.TransactionId, req.Cmd, req.DataLength, string(req.Data))
		resp, hasResponse := batch.responses[id]
		if hasResponse {
			log.Printf("Verify: Got response: %v %v %v %v\n", resp.TransactionId, resp.Cmd, resp.DataLength, string(resp.Data))
			log.Printf("Transaction %v has a request and response\n", id)
			num, err := resp.ParseResponseCode()
			if err != nil {
				panic(fmt.Sprintf("Could not parse response code for transaction %v", id))
			} else {
				if num == 200 {
					log.Printf("Transaction %v successfully verified.\n", id)
					return true
				}
			}
		}
	}
	log.Printf("Transaction %v could not be verified successfully!\n", id)
	return false
}

// VerifyTransactionAll goes through all requests and runs VerifyTransaction on all of them.
// Returns false if any one of the transactions could not be verified, otherwise true.
func (batch *RelpBatch) VerifyTransactionAll() bool {
	log.Printf("Verifying ALL transactions\n")
	for id := range batch.requests {
		verified := batch.VerifyTransaction(id)
		if !verified {
			return false
		}
	}
	return true
}

// RetryRequest retries sending the relp request frame by pushing it back
// to the work queue
func (batch *RelpBatch) RetryRequest(id uint64) {
	log.Printf("Retrying: Pushing request %v back to work queue", id)
	_, ok := batch.requests[id]
	if ok {
		batch.workQueue.PushBack(id)
	}
}

// RetryAllFailed verifies all transactions, and adds all the failed-to-verify requests back
// to the work queue
func (batch *RelpBatch) RetryAllFailed() {
	log.Printf("Verifying ALL transactions and retrying failed ones\n")
	for id := range batch.requests {
		verified := batch.VerifyTransaction(id)
		if !verified {
			batch.RetryRequest(id)
		}
	}
}

// GetWorkQueueLen gets the amount of requests in the work queue
func (batch *RelpBatch) GetWorkQueueLen() int {
	return batch.workQueue.Len()
}

// PopWorkQueue gets the front element from the work queue,
// deletes it from the queue and returns the ID for that request frame
func (batch *RelpBatch) PopWorkQueue() uint64 {
	elem := batch.workQueue.Front()
	id := elem.Value.(uint64)
	batch.workQueue.Remove(elem)
	return id
}
