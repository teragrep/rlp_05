package RelpFrame

// RelpFrame is the base struct for response and request frame structs
type RelpFrame struct {
	TransactionId uint64
	Cmd           string
	DataLength    int
	Data          []byte
}
