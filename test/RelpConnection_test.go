package test

import (
	"crypto/tls"
	"fmt"
	"github.com/teragrep/rlp_05/internal/RelpCommand"
	"github.com/teragrep/rlp_05/pkg/RelpBatch"
	"github.com/teragrep/rlp_05/pkg/RelpConnection"
	RelpDialer2 "github.com/teragrep/rlp_05/pkg/RelpDialer"
	RelpFrame2 "github.com/teragrep/rlp_05/pkg/RelpFrame"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestSingleMessage: Sends OPEN->SYSLOG->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestSingleMessage(t *testing.T) {
	relpServer := initServerConnection(false)
	time.Sleep(time.Second)
	// server ok, actual test

	sess := RelpConnection.RelpConnection{RelpDialer: &RelpDialer2.RelpPlainDialer{}}
	sess.Init()
	ok, _ := sess.Connect("127.0.0.1", 1601)

	if !ok {
		t.Errorf("Connection was not successful! (success=%v); want true", ok)
	}

	msgBatch := RelpBatch.RelpBatch{}
	msgBatch.Init()
	msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
		Cmd:        RelpCommand.RELP_SYSLOG,
		DataLength: len([]byte("HelloThisIsAMessage")),
		Data:       []byte("HelloThisIsAMessage"),
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
	if sess.Window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.Window.Size())
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

// TestMultipleMessage: Sends OPEN->SYSLOG->SYSLOG->SYSLOG->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestMultipleMessage(t *testing.T) {
	relpServer := initServerConnection(false)
	time.Sleep(time.Second)
	// server ok, actual test
	sess := RelpConnection.RelpConnection{RelpDialer: &RelpDialer2.RelpPlainDialer{}}
	sess.Init()
	ok, _ := sess.Connect("127.0.0.1", 1601)

	if !ok {
		t.Errorf("Connection was not successful! (success=%v); want true", ok)
	}

	for i := 0; i < 3; i++ {
		syslogMsg := []byte("HelloThisIsAMessage" + strconv.FormatInt(int64(i), 10))
		syslogMsgLen := len(syslogMsg)
		msgBatch := RelpBatch.RelpBatch{}
		msgBatch.Init()
		msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
			Cmd:        RelpCommand.RELP_SYSLOG,
			DataLength: syslogMsgLen,
			Data:       syslogMsg,
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
	if sess.Window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.Window.Size())
	}

	// kill server
	err := relpServer.Process.Kill()
	if err != nil {
		t.Error("Could not kill server\n")
	}
}

// TestMultipleMessageTLS: Sends OPEN->SYSLOG->SYSLOG->SYSLOG->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Uses TLS encrypted connection.
// Meaning all pending operations have been verified
func TestMultipleMessageTLS(t *testing.T) {
	relpServer := initServerConnection(true)
	time.Sleep(time.Second)
	// server ok, actual test
	sess := RelpConnection.RelpConnection{RelpDialer: &RelpDialer2.RelpTLSDialer{}}
	sess.Init()
	sess.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	ok, _ := sess.Connect("127.0.0.1", 1601)

	if !ok {
		t.Errorf("Connection was not successful! (success=%v); want true", ok)
	}

	for i := 0; i < 3; i++ {
		syslogMsg := []byte("HelloThisIsAMessage" + strconv.FormatInt(int64(i), 10))
		syslogMsgLen := len(syslogMsg)
		msgBatch := RelpBatch.RelpBatch{}
		msgBatch.Init()
		msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
			Cmd:        RelpCommand.RELP_SYSLOG,
			DataLength: syslogMsgLen,
			Data:       syslogMsg,
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
	if sess.Window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.Window.Size())
	}

	// kill server
	err := relpServer.Process.Kill()
	if err != nil {
		t.Error("Could not kill server\n")
	}
}

// TestMultiMessageBatch: Sends OPEN->SYSLOG(3x)->SYSLOG->SYSLOG(3x)->CLOSE messages.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestMultiMessageBatch(t *testing.T) {
	relpServer := initServerConnection(false)
	time.Sleep(time.Second)
	// server ok, actual test
	sess := RelpConnection.RelpConnection{RelpDialer: &RelpDialer2.RelpPlainDialer{}}
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
		msgBatch := RelpBatch.RelpBatch{}
		msgBatch.Init()
		msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
			Cmd:        RelpCommand.RELP_SYSLOG,
			DataLength: syslogMsgLen,
			Data:       syslogMsg,
		}})

		// put 3 messages on batches 0 and 2, and 1 message on batch 1
		if i != 1 {
			msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
				Cmd:        RelpCommand.RELP_SYSLOG,
				DataLength: syslogMsgLen,
				Data:       syslogMsg,
			}})
			msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
				Cmd:        RelpCommand.RELP_SYSLOG,
				DataLength: syslogMsgLen,
				Data:       syslogMsg,
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
	if sess.Window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.Window.Size())
	}

	// kill server
	err := relpServer.Process.Kill()
	if err != nil {
		t.Error("Could not kill server\n")
	}
}

// TestMultiMessageBatchWithDisconnect: Sends OPEN->SYSLOG(3x)->SYSLOG->SYSLOG(3x)->CLOSE messages,
// with a server disconnect in between.
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestMultiMessageBatchWithDisconnect(t *testing.T) {
	relpServer := initServerConnection(false)
	time.Sleep(time.Second)

	// server ok, actual test
	sess := RelpConnection.RelpConnection{RelpDialer: &RelpDialer2.RelpPlainDialer{}}
	sess.Init()
	retryRelpConnection(&sess)

	for i := 0; i < 3; i++ {
		syslogMsg := []byte("HelloThisIsAMessage" + strconv.FormatInt(int64(i), 10))
		syslogMsgLen := len(syslogMsg)
		msgBatch := RelpBatch.RelpBatch{}
		msgBatch.Init()
		msgBatch.PutRequest(&RelpFrame2.TX{RelpFrame2.Frame{
			Cmd:        RelpCommand.RELP_SYSLOG,
			DataLength: syslogMsgLen,
			Data:       syslogMsg,
		}})

		// kill server after first batch for 2 seconds
		if i == 1 {
			go func() {
				err := relpServer.Process.Kill()
				if err != nil {
					t.Errorf("Could not kill server\n")
				}
				time.Sleep(2 * time.Second)
				relpServer = initServerConnection(false)
			}()
		}

		// put 3 messages on batches 0 and 2, and 1 message on batch 1
		if i != 1 {
			msgBatch.PutRequest(&RelpFrame2.TX{RelpFrame2.Frame{
				Cmd:        RelpCommand.RELP_SYSLOG,
				DataLength: syslogMsgLen,
				Data:       syslogMsg,
			}})
			msgBatch.PutRequest(&RelpFrame2.TX{RelpFrame2.Frame{
				Cmd:        RelpCommand.RELP_SYSLOG,
				DataLength: syslogMsgLen,
				Data:       syslogMsg,
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
	if sess.Window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.Window.Size())
	}

	// kill server
	err2 := relpServer.Process.Kill()
	if err2 != nil {
		t.Errorf("Could not kill server\n")
	}
	fmt.Println("done")
}

// TestMultiMessageBatchWithDisconnectTLS: Sends OPEN->SYSLOG(3x)->SYSLOG->SYSLOG(3x)->CLOSE messages,
// with a server disconnect in between using encrypted TLS connection
// Checks for window (pending) to be empty and also that the batch's workQueue is empty.
// Meaning all pending operations have been verified
func TestMultiMessageBatchWithDisconnectTLS(t *testing.T) {
	relpServer := initServerConnection(true)
	time.Sleep(time.Second)

	// server ok, actual test
	sess := RelpConnection.RelpConnection{RelpDialer: &RelpDialer2.RelpTLSDialer{}}
	sess.Init()
	sess.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	retryRelpConnection(&sess)

	for i := 0; i < 3; i++ {
		syslogMsg := []byte("HelloThisIsAMessage" + strconv.FormatInt(int64(i), 10))
		syslogMsgLen := len(syslogMsg)
		msgBatch := RelpBatch.RelpBatch{}
		msgBatch.Init()
		msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
			Cmd:        RelpCommand.RELP_SYSLOG,
			DataLength: syslogMsgLen,
			Data:       syslogMsg,
		}})

		// kill server after first batch for 2 seconds
		if i == 1 {
			go func() {
				err := relpServer.Process.Kill()
				if err != nil {
					t.Errorf("Could not kill server\n")
				}
				time.Sleep(2 * time.Second)
				relpServer = initServerConnection(true)
			}()
		}

		// put 3 messages on batches 0 and 2, and 1 message on batch 1
		if i != 1 {
			msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
				Cmd:        RelpCommand.RELP_SYSLOG,
				DataLength: syslogMsgLen,
				Data:       syslogMsg,
			}})
			msgBatch.PutRequest(&RelpFrame2.TX{Frame: RelpFrame2.Frame{
				Cmd:        RelpCommand.RELP_SYSLOG,
				DataLength: syslogMsgLen,
				Data:       syslogMsg,
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
	if sess.Window.Size() != 0 {
		t.Errorf("RelpConnection.Window was not empty! (size=%v); want 0", sess.Window.Size())
	}

	// kill server
	err2 := relpServer.Process.Kill()
	if err2 != nil {
		t.Errorf("Could not kill server\n")
	}
	fmt.Println("done")
}

// Utils for testing

// retryRelpConnection disconnects and attempts to reconnect to the server every 5 seconds until succeeds
func retryRelpConnection(relpSess *RelpConnection.RelpConnection) {
	relpSess.TearDown()
	var cSuccess bool
	var cErr error
	cSuccess, cErr = relpSess.Connect("127.0.0.1", 1601)
	for !cSuccess || cErr != nil {
		log.Println(cErr)
		relpSess.TearDown()
		time.Sleep(5 * time.Second)
		cSuccess, cErr = relpSess.Connect("127.0.0.1", 1601)
	}
}

// initServerConnection initializes the relp server using a Java-based relp server
// the java demo relp server is hardcoded to run on 127.0.0.1:1601
func initServerConnection(tlsMode bool) *exec.Cmd {
	ready := make(chan int, 1)

	// get relp server jar
	wd, _ := os.Getwd()
	par := filepath.Dir(wd)
	for !strings.HasSuffix(par, "/rlp_05") {
		par = filepath.Dir(par)
	}
	jarLocation := par + "/resources/relp-server/java-relp-server-demo-jar-with-dependencies.jar"

	if _, err := os.Stat(jarLocation); err != nil {
		panic("No relp server jar found in " + jarLocation)
	}
	log.Printf("get relp server jar %v\n", jarLocation)

	// run it
	var relpServer *exec.Cmd
	if tlsMode {
		relpServer = exec.Command("/usr/bin/java",
			"-jar", jarLocation, "tls=true")
	} else {
		relpServer = exec.Command("/usr/bin/java",
			"-jar", jarLocation)
	}

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
