package main

import (
	"crypto/tls"
	"fmt"
	"github.com/teragrep/rlp_05/pkg/RelpBatch"
	"github.com/teragrep/rlp_05/pkg/RelpConnection"
	"github.com/teragrep/rlp_05/pkg/RelpDialer"
	RelpFrame2 "github.com/teragrep/rlp_05/pkg/RelpFrame"
	"log"
	"time"
)

// Usage example
var port = 1601

func main() {
	relpSess := RelpConnection.RelpConnection{RelpDialer: &RelpDialer.RelpTLSDialer{}}
	relpSess.Init()
	relpSess.TlsConfig = &tls.Config{InsecureSkipVerify: true}
	batch := RelpBatch.RelpBatch{}
	batch.Init()
	batch.PutRequest(&RelpFrame2.TX{
		Frame: RelpFrame2.Frame{
			Cmd:        "syslog",
			DataLength: len([]byte("HelloWorld")),
			Data:       []byte("HelloWorld"),
		},
	})

	notDone := true

	retry(&relpSess)
	fmt.Println("Continuing committing after 5 sec")
	time.Sleep(5 * time.Second)
	for notDone {
		commitErr := relpSess.Commit(&batch)
		if commitErr != nil {
			log.Printf("Error committing batch: '%v'\n", commitErr.Error())
		}

		if !batch.VerifyTransactionAll() {
			batch.RetryAllFailed()
			retry(&relpSess)
		} else {
			notDone = false
		}
	}

	relpSess.Disconnect()

	fmt.Println(">>DONE<<")
	// await for input, so the program doesn't exit
	a := 0.0
	fmt.Scanf("%f", &a)
}

func retry(relpSess *RelpConnection.RelpConnection) {
	relpSess.TearDown()
	var cSuccess bool
	var cErr error
	cSuccess, cErr = relpSess.Connect("127.0.0.1", port)
	for !cSuccess || cErr != nil {
		fmt.Println(cErr.Error())
		relpSess.TearDown()
		time.Sleep(5 * time.Second)
		cSuccess, cErr = relpSess.Connect("127.0.0.1", port)
	}
}
