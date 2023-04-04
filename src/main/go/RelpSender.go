package main

// RelpSender is the interface for all the methods RELP connections
// should contain
type RelpSender interface {
	Connect(hostname string, port int)
	Commit(batch *RelpBatch)
	Disconnect()
}
