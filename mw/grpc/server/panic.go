package server

import (
	"context"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
)

func GrpcServerPanicCapture() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		hasPanic := true
		defer func() {
			if !hasPanic {
				return
			}
			if r := recover(); r != nil {
				err = errs.ErrPanic(r)
				log.ZPanic(ctx, "rpc server panic", err, "method", info.FullMethod, "req", req)
			}
		}()
		resp, err = handler(ctx, req)
		hasPanic = false
		return
	})
}
