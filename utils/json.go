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
	if m, ok := v.(proto.Message); ok {
		return protoMarshalOptions.Marshal(m)
	}
	return json.Marshal(v)
}

func JsonUnmarshal(b []byte, v any) error {
	if m, ok := v.(proto.Message); ok {
		return protoUnmarshalOptions.Unmarshal(b, m)
	}
	return json.Unmarshal(b, v)
}
