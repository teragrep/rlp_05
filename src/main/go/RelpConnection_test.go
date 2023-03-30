package main

import (
	"fmt"
	"strconv"
	"testing"
)

// TestConnection: Sends OPEN->SYSLOG->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestConnection(t *testing.T) {
	sess := RelpConnection{}
	sess.Init()
	ok, _ := sess.Connect("127.0.0.1", 1601)

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

// TestConnectionMulti: Sends OPEN->SYSLOG->SYSLOG->SYSLOG->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestConnectionMulti(t *testing.T) {
	sess := RelpConnection{}
	sess.Init()
	ok, _ := sess.Connect("127.0.0.1", 1601)

	if !ok {
		t.Errorf("Connection was not successful! (success=%v); want true", ok)
	}

	for i := 0; i < 3; i++ {
		syslogMsg := []byte("HelloThisIsAMessage" + strconv.FormatInt(int64(i), 10))
		syslogMsgLen := len(syslogMsg)
		msgBatch := RelpBatch{}
		msgBatch.Init()
		msgBatch.PutRequest(&RelpFrameTX{RelpFrame{
			cmd:        RELP_SYSLOG,
			dataLength: syslogMsgLen,
			data:       syslogMsg,
		}})

		sess.Commit(&msgBatch)

		// batch queue empty
		if msgBatch.GetWorkQueueLen() != 0 {
			t.Errorf("RelpBatch.WorkQueue was not empty! (len=%v); want 0", msgBatch.GetWorkQueueLen())
		}
	}

	disOk := sess.Disconnect()

	if !disOk {
		t.Errorf("Disconnection was not successful! (success=%v); want true", disOk)
	}

	// no stuff pending
	if sess.window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.window.Size())
	}
}

// TestConnectionMulti: Sends OPEN->SYSLOG(m.)->SYSLOG->SYSLOG(m.)->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestConnectionMultiBatch(t *testing.T) {
	sess := RelpConnection{}
	sess.Init()
	ok, _ := sess.Connect("127.0.0.1", 1601)

	if !ok {
		t.Errorf("Connection was not successful! (success=%v); want true", ok)
	}

	//fmt.Println("Waiting 5 secs before continuing")
	//time.Sleep(5 * time.Second)

	for i := 0; i < 3; i++ {
		syslogMsg := []byte("HelloThisIsAMessage" + strconv.FormatInt(int64(i), 10))
		syslogMsgLen := len(syslogMsg)
		msgBatch := RelpBatch{}
		msgBatch.Init()
		msgBatch.PutRequest(&RelpFrameTX{RelpFrame{
			cmd:        RELP_SYSLOG,
			dataLength: syslogMsgLen,
			data:       syslogMsg,
		}})

		// put 3 messages on batches 0 and 2, and 1 message on batch 1
		if i != 1 {
			msgBatch.PutRequest(&RelpFrameTX{RelpFrame{
				cmd:        RELP_SYSLOG,
				dataLength: syslogMsgLen,
				data:       syslogMsg,
			}})
			msgBatch.PutRequest(&RelpFrameTX{RelpFrame{
				cmd:        RELP_SYSLOG,
				dataLength: syslogMsgLen,
				data:       syslogMsg,
			}})
		}

		fmt.Printf("Got workQ: %v\n", msgBatch.GetWorkQueueLen())

		sess.Commit(&msgBatch)

		// batch queue empty
		if msgBatch.GetWorkQueueLen() != 0 {
			t.Errorf("RelpBatch.WorkQueue was not empty! (len=%v); want 0", msgBatch.GetWorkQueueLen())
		}
	}

	disOk := sess.Disconnect()

	if !disOk {
		t.Errorf("Disconnection was not successful! (success=%v); want true", disOk)
	}

	// no stuff pending
	if sess.window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.window.Size())
	}
}
