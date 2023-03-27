package main

import (
	"fmt"
)

// TODO: Process ACKs in RelpConnection (needs RelpParser?)

func main() {
	relpSess := RelpConnection{}
	relpSess.Init()
	relpSess.Connect("127.0.0.1", 1601)

	batch := RelpBatch{}
	batch.Init()

	batch.PutRequest(&RelpFrameTX{
		RelpFrame{
			transactionId: 1337,
			cmd:           "syslog",
			dataLength:    10,
			data:          []byte("HelloWorld"),
		},
	})

	relpSess.Commit(&batch)
	//relpSess.SendBatch(&batch)
	relpSess.Disconnect()

	// await for input, so the program doesn't exit
	a := 0.0
	fmt.Scanf("%f", &a)
}
