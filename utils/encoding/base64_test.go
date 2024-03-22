// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
