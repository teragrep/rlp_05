package main

import "errors"

type RelpWindow struct {
	pending map[uint64]uint64
}

func (win *RelpWindow) Init() *RelpWindow {
	win.pending = make(map[uint64]uint64)
	return win
}

func (win *RelpWindow) PutPending(txnId, reqId uint64) {
	win.pending[txnId] = reqId
}

func (win *RelpWindow) IsPending(txnId uint64) bool {
	_, ok := win.pending[txnId]
	return ok
}

func (win *RelpWindow) GetPending(txnId uint64) (uint64, error) {
	v, ok := win.pending[txnId]
	if ok {
		return v, nil
	} else {
		return 0, errors.New("txnId did not have a matching value in RelpWindow")
	}
}

func (win *RelpWindow) RemovePending(txnId uint64) {
	delete(win.pending, txnId)
}

func (win *RelpWindow) Size() int {
	return len(win.pending)
}
