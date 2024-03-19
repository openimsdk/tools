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
