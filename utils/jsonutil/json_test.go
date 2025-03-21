// Copyright © 2024 OpenIM open source community. All rights reserved.
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

package jsonutil

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TODO: If you also want tests that include protobuf messages, define a.proto file first, and then use the protoc command to generate the Go code

func TestJsonMarshal(t *testing.T) {
	structData := struct{ Name string }{"John"}
	structBytes, err := JsonMarshal(structData)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"Name":"John"}`, string(structBytes))

	marshalerData := json.RawMessage(`{"type":"raw"}`)
	marshalerBytes, err := JsonMarshal(marshalerData)
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"raw"}`, string(marshalerBytes))
}

func TestJsonUnmarshal(t *testing.T) {
	structBytes := []byte(`{"Name":"Jane"}`)
	var structData struct{ Name string }
	err := JsonUnmarshal(structBytes, &structData)
	assert.NoError(t, err)
	assert.Equal(t, "Jane", structData.Name)

	marshalerBytes := []byte(`{"type":"unmarshal"}`)
	var marshalerData json.RawMessage
	err = JsonUnmarshal(marshalerBytes, &marshalerData)
	assert.NoError(t, err)
	assert.Equal(t, `{"type":"unmarshal"}`, string(marshalerData))
}
