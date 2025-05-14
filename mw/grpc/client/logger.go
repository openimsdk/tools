package client

import (
	"context"
	"time"

	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
)

func GrpcClientLogger() grpc.DialOption {
	return grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		log.ZInfo(ctx, "rpc client request", "method", method, "req", req)
		start := time.Now()
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			log.ZInfo(ctx, "rpc client response success", "method", method, "cost", time.Since(start), "req", req, "resp", reply)
		} else {
			log.ZError(ctx, "rpc client response error", err, "method", method, "cost", time.Since(start), "req", req)
		}
		return err
	})
}
