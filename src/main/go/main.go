package main

import (
	"crypto/tls"
	"fmt"
	"log"
	"time"
)

// Usage example
var port = 1601

func main() {
	relpSess := RelpConnection{RelpDialer: &RelpTLSDialer{}}
	relpSess.Init()
	relpSess.tlsConfig = &tls.Config{InsecureSkipVerify: true}
	batch := RelpBatch{}
	batch.Init()
	batch.PutRequest(&RelpFrameTX{
		RelpFrame{
			cmd:        "syslog",
			dataLength: len([]byte("HelloWorld")),
			data:       []byte("HelloWorld"),
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

func retry(relpSess *RelpConnection) {
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
