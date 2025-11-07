package encoding

import "testing"

func TestBase64EncodeDecode(t *testing.T) {
	originalText := "This is a test string!"

	// Encode
	encoded := Base64Encode(originalText)
	if encoded == "" {
		t.Errorf("Base64Encode returned an empty string")
	}

	// Decode
	decoded, err := Base64Decode(encoded)
	if err != nil {
		t.Errorf("Base64Decode returned an error: %v", err)
	}

	if decoded != originalText {
		t.Errorf("Base64Decode = %v, want %v", decoded, originalText)
	}
}

func TestBase64DecodeErrorHandling(t *testing.T) {
	malformedBase64 := "This is not base64!"
	_, err := Base64Decode(malformedBase64)
	if err == nil {
		t.Errorf("Expected an error for malformed base64 input, but none was returned")
	}
}
