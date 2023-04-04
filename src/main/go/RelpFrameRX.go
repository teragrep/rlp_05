package main

import (
	"strconv"
)

// RelpFrameRX is a struct containing the response frame
type RelpFrameRX struct {
	RelpFrame
}

// ParseResponseCode parses the response code as an integer from the request frame.
// If parsing can't be done, returns 0 as the code and an error
func (rxFrame *RelpFrameRX) ParseResponseCode() (int, error) {
	var code = make([]byte, 3, 3)
	for i, v := range rxFrame.data {
		if i == 3 && v == ' ' {
			num, err := strconv.ParseInt(string(code), 10, 64)
			if err != nil {
				return 0, &ResponseCodeParsingError{reason: "error parsing number from string to int64"}
			} else {
				return int(num), nil
			}
		} else if i >= 3 {
			return 0, &ResponseCodeParsingError{reason: "response code was longer than 3 numbers; want <= 3"}
		}

		if v >= 48 && v <= 57 {
			//0-9 ascii
			code[i] = v
		} else {
			return 0, &ResponseCodeParsingError{reason: "encountered non-number ASCII char in response code"}
		}
	}

	return 0, &ResponseCodeParsingError{reason: "response code could not been found"}
}
