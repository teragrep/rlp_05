package RelpFrame

import (
	"github.com/teragrep/rlp_05/Errors"
	"strconv"
)

// RX RelpFrameRX is a struct containing the response frame
type RX struct {
	RelpFrame
}

// ParseResponseCode parses the response code as an integer from the request frame.
// If parsing can't be done, returns 0 as the code and an error
func (rxFrame *RX) ParseResponseCode() (int, error) {
	var code = make([]byte, 3, 3)
	for i, v := range rxFrame.Data {
		if i == 3 && v == ' ' {
			num, err := strconv.ParseInt(string(code), 10, 64)
			if err != nil {
				return 0, &Errors.ResponseCodeParsingError{Reason: "error parsing number from string to int64"}
			} else {
				return int(num), nil
			}
		} else if i >= 3 {
			return 0, &Errors.ResponseCodeParsingError{Reason: "response code was longer than 3 numbers; want <= 3"}
		}

		if v >= 48 && v <= 57 {
			//0-9 ascii
			code[i] = v
		} else {
			return 0, &Errors.ResponseCodeParsingError{Reason: "encountered non-number ASCII char in response code"}
		}
	}

	return 0, &Errors.ResponseCodeParsingError{Reason: "response code could not been found"}
}
