package main

type RelpSender interface {
	connect(hostname string, port int)
	commit(batch RelpBatch)
	disconnect()
}
