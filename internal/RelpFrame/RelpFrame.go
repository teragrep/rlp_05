package RelpFrame

// Frame RelpFrame is the base struct for response and request frame structs
type Frame struct {
	TransactionId uint64
	Cmd           string
	DataLength    int
	Data          []byte
}
