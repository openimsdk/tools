package idutil

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestGetMsgIDByMD5 checks if GetMsgIDByMD5 returns a valid MD5 hash string.
func TestGetMsgIDByMD5(t *testing.T) {
	sendID := "12345"
	msgID := GetMsgIDByMD5(sendID)

	// MD5 hash is a 32 character hexadecimal number.
	assert.Regexp(t, regexp.MustCompile("^[a-fA-F0-9]{32}$"), msgID, "The returned msgID should be a valid MD5 hash")
}

// TestOperationIDGenerator checks if OperationIDGenerator returns a string that looks like a valid operation ID.
func TestOperationIDGenerator(t *testing.T) {
	opID := OperationIDGenerator()

	// The operation ID should be a long number (timestamp + random number).
	// Just check if it is numeric and has a reasonable length.
	assert.Regexp(t, regexp.MustCompile("^[0-9]{13,}$"), opID, "The returned operation ID should be numeric and long")
}
