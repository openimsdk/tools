package server

import (
	"context"
	"time"

	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
)

func GrpcServerLogger() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		log.ZInfo(ctx, "rpc server request", "method", info.FullMethod, "req", req)
		start := time.Now()
		resp, err = handler(ctx, req)
		if err == nil {
			log.ZInfo(ctx, "rpc server response success", "method", info.FullMethod, "cost", time.Since(start), "req", req, "resp", resp)
		} else {
			log.ZError(ctx, "rpc server response error", err, "method", info.FullMethod, "cost", time.Since(start), "req", req)
		}
		return
	})
}
