package standalone

import (
	"context"
	"time"

	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func newClientConn() *clientConn {
	return &clientConn{
		registry:   newRegistry(),
		serializer: newProtoSerializer(),
	}
}

type clientConn struct {
	registry   *registry
	serializer serializer
}

func (x *clientConn) Invoke(ctx context.Context, method string, args any, reply any, opts ...grpc.CallOption) error {
	handler, err := x.registry.getMethod(ctx, method)
	if err != nil {
		return err
	}
	log.ZInfo(ctx, "standalone rpc server request", "method", method, "req", args)
	start := time.Now()
	resp, err := handler(ctx, args, nil)
	if err == nil {
		log.ZInfo(ctx, "standalone rpc server response success", "method", method, "cost", time.Since(start), "req", args, "resp", resp)
	} else {
		log.ZError(ctx, "standalone rpc server response error", err, "method", method, "cost", time.Since(start), "req", args)
	}
	if err != nil {
		return err
	}
	tmp, err := x.serializer.Marshal(resp)
	if err != nil {
		return err
	}
	return x.serializer.Unmarshal(tmp, reply)
}

func (x *clientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Errorf(codes.Unimplemented, "method stream not implemented")
}
