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

package utils

import (
	"encoding/json"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	protoMarshalOptions = protojson.MarshalOptions{
		AllowPartial:    true,
		UseProtoNames:   true,
		UseEnumNumbers:  true,
		EmitUnpopulated: true,
	}
	protoUnmarshalOptions = protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: true,
	}
)

func JsonMarshal(v any) ([]byte, error) {
	switch o := v.(type) {
	case json.Marshaler:
		return o.MarshalJSON()
	case proto.Message:
		return protoMarshalOptions.Marshal(o)
	default:
		return json.Marshal(o)
	}
}

func JsonUnmarshal(b []byte, v any) error {
	switch o := v.(type) {
	case json.Unmarshaler:
		return o.UnmarshalJSON(b)
	case proto.Message:
		return protoUnmarshalOptions.Unmarshal(b, o)
	default:
		return json.Unmarshal(b, v)
	}
}
