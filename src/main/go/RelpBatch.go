package main

import (
	"container/list"
	"errors"
)

type RelpBatch struct {
	requests  map[uint64]RelpFrameTX
	responses map[uint64]RelpFrameRX
	workQueue *list.List
	requestId uint64
}

func (batch *RelpBatch) Init() {
	batch.requests = make(map[uint64]RelpFrameTX)
	batch.responses = make(map[uint64]RelpFrameRX)
	batch.workQueue = list.New()
	batch.requestId = 0
}

func (batch *RelpBatch) Insert(syslogMsg []byte) uint64 {
	relpRequest := RelpFrameTX{
		RelpFrame{
			data:       syslogMsg,
			dataLength: len(syslogMsg),
			cmd:        "syslog",
		},
	}

	return batch.PutRequest(&relpRequest)
}

func (batch *RelpBatch) PutRequest(tx *RelpFrameTX) uint64 {
	batch.requestId += 1
	batch.requests[batch.requestId] = *tx
	batch.workQueue.PushBack(batch.requestId)

	return batch.requestId
}

func (batch *RelpBatch) GetRequest(id uint64) (*RelpFrameTX, error) {
	v, ok := batch.requests[id]
	if ok {
		return &v, nil
	} else {
		return nil, errors.New("could not find request in batch")
	}
}

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

func (batch *RelpBatch) GetResponse(id uint64) (*RelpFrameRX, error) {
	v, ok := batch.responses[id]
	if ok {
		return &v, nil
	} else {
		return nil, errors.New("could not get response from batch")
	}
}

func (batch *RelpBatch) PutResponse(id uint64, response *RelpFrameRX) {
	_, ok := batch.requests[id]
	if ok {
		batch.responses[id] = *response
	}
}

func (batch *RelpBatch) VerifyTransaction(id uint64) bool {
	_, hasRequest := batch.requests[id]
	if hasRequest {
		v, hasResponse := batch.responses[id]
		if hasResponse {
			num, err := v.ParseResponseCode()
			if err != nil {
				panic("Could not parse response code from response!")
			} else {
				if num == 200 {
					return true
				}
			}
		}
	}
	return false
}

func (batch *RelpBatch) VerifyTransactionAll() bool {
	for id := range batch.requests {
		verified := batch.VerifyTransaction(id)
		if !verified {
			return false
		}
	}
	return true
}

func (batch *RelpBatch) RetryRequest(id uint64) {
	_, ok := batch.requests[id]
	if ok {
		batch.workQueue.PushBack(id)
	}
}

func (batch *RelpBatch) RetryAllFailed() {
	for id := range batch.requests {
		verified := batch.VerifyTransaction(id)
		if !verified {
			batch.RetryRequest(id)
		}
	}
}

func (batch *RelpBatch) GetWorkQueueLen() int {
	return batch.workQueue.Len()
}

func (batch *RelpBatch) PopWorkQueue() uint64 {
	elem := batch.workQueue.Front()
	id := elem.Value.(uint64)
	batch.workQueue.Remove(elem)
	return id
}
