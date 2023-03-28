package main

import "errors"

// RelpWindow is a struct that contains all the ids (transaction-request) mapped
// As the "pending" name suggests, they are the transactions still in progress
type RelpWindow struct {
	pending map[uint64]uint64
}

// Init initializes the pending map
func (win *RelpWindow) Init() *RelpWindow {
	win.pending = make(map[uint64]uint64)
	return win
}

// PutPending inserts an id-id pair to the map
func (win *RelpWindow) PutPending(txnId, reqId uint64) {
	win.pending[txnId] = reqId
}

// IsPending checks if a transaction is pending
func (win *RelpWindow) IsPending(txnId uint64) bool {
	_, ok := win.pending[txnId]
	return ok
}

// GetPending gets the requestId for the specified transactionId
func (win *RelpWindow) GetPending(txnId uint64) (uint64, error) {
	v, ok := win.pending[txnId]
	if ok {
		return v, nil
	} else {
		return 0, errors.New("txnId did not have a matching value in RelpWindow")
	}
}

// RemovePending removes a pending transaction from the map
func (win *RelpWindow) RemovePending(txnId uint64) {
	delete(win.pending, txnId)
}

// Size returns the amount of pending ids in the map
func (win *RelpWindow) Size() int {
	return len(win.pending)
}
