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
	"github.com/openimsdk/tools/errs"

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
		m, err := o.MarshalJSON()
		return m, errs.Wrap(err)
	case proto.Message:
		m, err := protoMarshalOptions.Marshal(o)
		return m, errs.Wrap(err)
	default:
		m, err := json.Marshal(o)
		return m, err
	}
}

func JsonUnmarshal(b []byte, v any) error {
	switch o := v.(type) {
	case json.Unmarshaler:
		return errs.Wrap(o.UnmarshalJSON(b))
	case proto.Message:
		return errs.Wrap(protoUnmarshalOptions.Unmarshal(b, o))
	default:
		return errs.Wrap(json.Unmarshal(b, v))
	}
}

func StructToJsonString(param any) string {
	dataType, _ := JsonMarshal(param)
	dataString := string(dataType)
	return dataString
}

// The incoming parameter must be a pointer
func JsonStringToStruct(s string, args any) error {
	err := json.Unmarshal([]byte(s), args)
	return err
}
