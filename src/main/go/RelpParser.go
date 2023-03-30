package main

import (
	"bytes"
	"log"
	"strconv"
	"strings"
)

// constants, such as parser state (PS_ prefix) and max command length (MAX_CMD_LEN)
const (
	MAX_CMD_LEN = 11
	PS_TXN      = 0
	PS_CMD      = 1
	PS_LEN      = 2
	PS_DATA     = 3
	PS_NL       = 4
)

// RelpParser contains the fields necessary for completing the response (RX)
// parsing. The results of the parse operation can be found from the frameTxnId, frameCmdString, frameLen
// and frameData fields.
type RelpParser struct {
	state            int
	isComplete       bool
	frameTxnIdString string
	frameTxnId       uint64
	frameCmdString   string
	frameLenString   string
	frameLen         int
	frameLenLeft     int
	frameData        *bytes.Buffer
}

// Parse is used to parse the incoming response (RX).
// It will populate the RelpParser struct's fields with the parsed data
func (parser *RelpParser) Parse(b byte) {
	switch parser.state {
	case PS_TXN:
		{
			if b == ' ' {
				num, err := strconv.ParseUint(parser.frameTxnIdString, 10, 64)
				if err != nil {
					log.Fatalln("RelpParser: Could not parse frameTxnId from string")
				} else {
					parser.frameTxnId = num
					parser.state = PS_CMD
				}
			} else {
				parser.frameTxnIdString += string(b)
			}
		}
	case PS_CMD:
		{
			if b == ' ' {
				parser.state = PS_LEN
				// constraints
				if len(parser.frameCmdString) > MAX_CMD_LEN &&
					strings.Compare(parser.frameCmdString, RELP_OPEN) != 0 &&
					strings.Compare(parser.frameCmdString, RELP_CLOSE) != 0 &&
					strings.Compare(parser.frameCmdString, RELP_ABORT) != 0 &&
					strings.Compare(parser.frameCmdString, RELP_SERVER_CLOSE) != 0 &&
					strings.Compare(parser.frameCmdString, RELP_SYSLOG) != 0 &&
					strings.Compare(parser.frameCmdString, RELP_RSP) != 0 {
					panic("RelpParser: Invalid command")
				}
			} else {
				parser.frameCmdString += string(b)
			}
			break
		}
	case PS_LEN:
		{
			// librelp sometimes uses HEADER = TXNR SP CMD SP DATALEN LF instead of SP as last one
			if b == ' ' || b == '\n' {
				num, err := strconv.ParseInt(parser.frameLenString, 10, 64)
				if err != nil {
					log.Fatalln("RelpParser: Could not parse frame length from string")
				} else {
					parser.frameLen = int(num)
				}

				if parser.frameLen < 0 {
					panic("RelpParser: Frame length must be >= 0")
				}

				parser.frameLenLeft = parser.frameLen
				parser.frameData = bytes.NewBuffer(make([]byte, 0, parser.frameLen))

				// length bytes done, move to next stage
				if parser.frameLen == 0 {
					// no data
					parser.state = PS_NL
				} else {
					// data
					parser.state = PS_DATA
				}

				if b == '\n' {
					if parser.frameLen == 0 {
						parser.isComplete = true
					}
				}
			} else {
				parser.frameLenString += string(b)
			}
			break
		}
	case PS_DATA:
		{
			if parser.isComplete {
				parser.state = PS_NL
			}

			// only read frameLen of data
			if parser.frameLenLeft > 0 {
				parser.frameData.WriteByte(b)
				parser.frameLenLeft -= 1
			}

			if parser.frameLenLeft == 0 {
				// parsing done, no data left
				parser.state = PS_NL
			}
			break
		}
	case PS_NL:
		{
			if b == '\n' {
				// RELP msg always ends with NL
				parser.isComplete = true
				log.Printf("RelpParser: Parser complete. Got: %v %v %v %v\n",
					parser.frameTxnId, parser.frameCmdString, parser.frameLen, parser.frameData)
			} else {
				panic("RelpParser: Final byte was not NL")
			}
			break
		}
	default:
		{
			break
		}
	}
}
