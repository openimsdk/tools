package server

import (
	"context"

	"github.com/openimsdk/tools/checker"
	"google.golang.org/grpc"
)

func GrpcServerRequestValidate() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if err := checker.Validate(req); err != nil {
			return nil, err
		}
		return handler(ctx, req)
	})
}
