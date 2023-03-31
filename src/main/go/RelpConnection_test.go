package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestConnection: Sends OPEN->SYSLOG->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestConnection(t *testing.T) {
	relpServer := InitServerConnection()
	time.Sleep(time.Second)
	// server ok, actual test

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

	err := sess.Commit(&msgBatch)
	if err != nil {
		t.Errorf("Error committing batch (err!=nil); want nil")
	}
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

	// kill server
	err = relpServer.Process.Kill()
	if err != nil {
		t.Error("Could not kill server\n")
	}
}

// TestConnectionMulti: Sends OPEN->SYSLOG->SYSLOG->SYSLOG->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestConnectionMulti(t *testing.T) {
	relpServer := InitServerConnection()
	time.Sleep(time.Second)
	// server ok, actual test
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

		err := sess.Commit(&msgBatch)
		if err != nil {
			t.Errorf("Error committing batch (err!=nil); want nil")
		}

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

	// kill server
	err := relpServer.Process.Kill()
	if err != nil {
		t.Error("Could not kill server\n")
	}
}

// TestConnectionMulti: Sends OPEN->SYSLOG(m.)->SYSLOG->SYSLOG(m.)->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestConnectionMultiBatch(t *testing.T) {
	relpServer := InitServerConnection()
	time.Sleep(time.Second)
	// server ok, actual test
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

		err := sess.Commit(&msgBatch)
		if err != nil {
			t.Errorf("Error committing batch (err!=nil); want nil")
		}

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

	// kill server
	err := relpServer.Process.Kill()
	if err != nil {
		t.Error("Could not kill server\n")
	}
}

// TestConnectionHandleDisconnect: Sends OPEN->SYSLOG(m.)->SYSLOG->SYSLOG(m.)->CLOSE messages,
// with a server disconnect in between.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestConnectionHandleDisconnect(t *testing.T) {
	relpServer := InitServerConnection()
	time.Sleep(time.Second)

	// server ok, actual test
	sess := RelpConnection{}
	sess.Init()
	retryRelpConnection(&sess)

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

		// kill server after first batch for 2 seconds
		if i == 1 {
			go func() {
				err := relpServer.Process.Kill()
				if err != nil {
					t.Errorf("Could not kill server\n")
				}
				time.Sleep(2 * time.Second)
				relpServer = InitServerConnection()
			}()
		}

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

		notDone := true
		for notDone {
			commitErr := sess.Commit(&msgBatch)
			if commitErr != nil {
				log.Printf("Error committing batch: '%v'\n", commitErr.Error())
			}

			if !msgBatch.VerifyTransactionAll() {
				msgBatch.RetryAllFailed()
				retryRelpConnection(&sess)
			} else {
				notDone = false
			}
		}

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

	// kill server
	err2 := relpServer.Process.Kill()
	if err2 != nil {
		t.Errorf("Could not kill server\n")
	}
	fmt.Println("done")
}

// utils
func retryRelpConnection(relpSess *RelpConnection) {
	relpSess.TearDown()
	var cSuccess bool
	var cErr error
	cSuccess, cErr = relpSess.Connect("127.0.0.1", 1601)
	for !cSuccess || cErr != nil {
		relpSess.TearDown()
		time.Sleep(5 * time.Second)
		cSuccess, cErr = relpSess.Connect("127.0.0.1", 1601)
	}
}

func InitServerConnection() *exec.Cmd {
	ready := make(chan int, 1)

	// get relp server jar
	wd, _ := os.Getwd()
	par := filepath.Dir(wd)
	for !strings.HasSuffix(par, "/src") {
		par = filepath.Dir(par)
	}
	jarLocation := par + "/resources/relp-server/java-relp-server-demo-jar-with-dependencies.jar"

	if _, err := os.Stat(jarLocation); err != nil {
		panic("No relp server jar found in " + jarLocation)
	}
	log.Printf("get relp server jar %v\n", jarLocation)

	// run it
	relpServer := exec.Command("/usr/bin/java",
		"-jar", jarLocation)

	// start in coroutine
	go func() {
		srvErr := relpServer.Start()
		if srvErr != nil {
			panic("failed to start relp server")
		}
		ready <- 1
	}()
	<-ready
	return relpServer
}
