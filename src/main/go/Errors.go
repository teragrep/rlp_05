package main

import "fmt"

type ResponseParsingError struct {
	position string
	reason   string
}

func (rpe *ResponseParsingError) Error() string {
	return fmt.Sprintf("error occurred while parsing response at position %s, reason: %s",
		rpe.position, rpe.reason)
}

type ResponseCodeParsingError struct {
	reason string
}

func (rcpe *ResponseCodeParsingError) Error() string {
	return fmt.Sprintf("Error parsing response code: %s", rcpe.reason)
}

type AckReadingError struct {
	reason string
}

func (are *AckReadingError) Error() string {
	return fmt.Sprintf("ACK reading error: %s", are.reason)
}

type ConnectionEstablishmentError struct {
	hostname  string
	port      int
	reason    string
	encrypted bool
	protocol  string
}

func (cee *ConnectionEstablishmentError) Error() string {
	encryptedStr := "unencrypted"
	if cee.encrypted {
		encryptedStr = "encrypted"
	}
	return fmt.Sprintf("Could not establish %v connection to %v:%v using protocol %v for reason: %v",
		encryptedStr, cee.hostname, cee.port, cee.protocol, cee.reason)
}
