package Errors

import "fmt"

type ResponseParsingError struct {
	Position string
	Reason   string
}

func (rpe *ResponseParsingError) Error() string {
	return fmt.Sprintf("error occurred while parsing response at position %s, reason: %s",
		rpe.Position, rpe.Reason)
}

type ResponseCodeParsingError struct {
	Reason string
}

func (rcpe *ResponseCodeParsingError) Error() string {
	return fmt.Sprintf("Error parsing response code: %s", rcpe.Reason)
}

type AckReadingError struct {
	Reason string
}

func (are *AckReadingError) Error() string {
	return fmt.Sprintf("ACK reading error: %s", are.Reason)
}

type ConnectionEstablishmentError struct {
	Hostname  string
	Port      int
	Reason    string
	Encrypted bool
	Protocol  string
}

func (cee *ConnectionEstablishmentError) Error() string {
	encryptedStr := "unencrypted"
	if cee.Encrypted {
		encryptedStr = "encrypted"
	}
	return fmt.Sprintf("Could not establish %v connection to %v:%v using protocol %v for reason: %v",
		encryptedStr, cee.Hostname, cee.Port, cee.Protocol, cee.Reason)
}
