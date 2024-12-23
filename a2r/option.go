package a2r

import (
	"context"
	"github.com/openimsdk/tools/mw"
	"google.golang.org/grpc"
)

func NewNilReplaceOption[A, B, C any](_ func(client C, ctx context.Context, req *A, options ...grpc.CallOption) (*B, error)) *Option[A, B] {
	return &Option[A, B]{
		RespAfter: respNilReplace[B],
	}
}

// respNilReplace replaces nil maps and slices in the resp object and initializing them.
func respNilReplace[T any](data *T) error {
	mw.ReplaceNil(data)
	return nil
}
