package RelpWindow

import (
	"errors"
	"log"
)

// RelpWindow is a struct that contains all the ids (frame id->frame?) mapped
// As the "pending" name suggests, they are the transactions still in progress
type RelpWindow struct {
	Pending map[uint64]uint64
}

// Init initializes the pending map
func (win *RelpWindow) Init() *RelpWindow {
	win.Pending = make(map[uint64]uint64)
	return win
}

// PutPending inserts an id-id pair to the map
func (win *RelpWindow) PutPending(txnId, reqId uint64) {
	it, has := win.Pending[txnId]
	if has {
		log.Println("Pending had for txnId: ", txnId, it)
	}
	win.Pending[txnId] = reqId
}

// IsPending checks if a transaction is pending
func (win *RelpWindow) IsPending(txnId uint64) bool {
	_, ok := win.Pending[txnId]
	return ok
}

// GetPending gets the requestId for the specified transactionId
func (win *RelpWindow) GetPending(txnId uint64) (uint64, error) {
	v, ok := win.Pending[txnId]
	if ok {
		return v, nil
	} else {
		return 0, errors.New("txnId did not have a matching value in RelpWindow")
	}
}

// RemovePending removes a pending transaction from the map
func (win *RelpWindow) RemovePending(txnId uint64) {
	delete(win.Pending, txnId)
}

// Size returns the amount of pending ids in the map
func (win *RelpWindow) Size() int {
	return len(win.Pending)
}
