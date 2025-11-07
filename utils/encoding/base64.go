package encoding

import (
	"encoding/base64"

	"github.com/openimsdk/tools/errs"
)

// Base64Encode converts input string data to a Base64-encoded string.
func Base64Encode(data string) string {
	return base64.StdEncoding.EncodeToString([]byte(data))
}

// Base64Decode decodes a Base64-encoded string and returns the original data as a string.
// It returns an error if the input is not valid Base64.
func Base64Decode(data string) (string, error) {
	decodedBytes, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", errs.WrapMsg(err, "DecodeString failed", "data", data)
	}
	return string(decodedBytes), nil
}
