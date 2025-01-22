package standalone

import "google.golang.org/protobuf/proto"

type serializer interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
}

func newProtoSerializer() serializer {
	return protoSerializer{}
}

type protoSerializer struct{}

func (protoSerializer) Marshal(in any) ([]byte, error) {
	return proto.Marshal(in.(proto.Message))
}

func (protoSerializer) Unmarshal(b []byte, out any) error {
	return proto.Unmarshal(b, out.(proto.Message))
}
