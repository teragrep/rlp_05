package main

import (
	"testing"
)

// TestConnection: Sends OPEN->SYSLOG->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestConnection(t *testing.T) {
	sess := RelpConnection{}
	sess.Init()
	ok := sess.Connect("127.0.0.1", 1601)

	if !ok {
		t.Errorf("Connection was not successful! (success=%v); want true", ok)
	}

	msgBatch := RelpBatch{}
	msgBatch.Init()
	msgBatch.PutRequest(&RelpFrameTX{RelpFrame{
		cmd:        RELP_SYSLOG,
		dataLength: len([]byte("HelloThisIsAMessage")),
		data:       []byte("HelloThisIsAMessage"),
	}})

	sess.Commit(&msgBatch)
	disOk := sess.Disconnect()

	if !disOk {
		t.Errorf("Disconnection was not successful! (success=%v); want true", disOk)
	}

	// no stuff pending
	if sess.window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.window.Size())
	}

	// batch queue empty
	if msgBatch.GetWorkQueueLen() != 0 {
		t.Errorf("RelpBatch.WorkQueue was not empty! (len=%v); want 0", msgBatch.GetWorkQueueLen())
	}
}
