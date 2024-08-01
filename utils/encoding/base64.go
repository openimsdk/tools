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
