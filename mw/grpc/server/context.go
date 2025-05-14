package server

import (
	"context"
	"fmt"

	"github.com/openimsdk/protocol/constant"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func GrpcServerMetadataContext() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.New(codes.InvalidArgument, "missing metadata").Err()
		}
		if len(md.Get(constant.OperationID)) != 1 {
			return nil, status.New(codes.InvalidArgument, "operationID error").Err()
		}
		if keys := md.Get(constant.RpcCustomHeader); len(keys) > 0 {
			ctx = context.WithValue(ctx, constant.RpcCustomHeader, keys)
			for _, key := range keys {
				values := md.Get(key)
				if len(values) == 0 {
					return nil, status.New(codes.InvalidArgument, fmt.Sprintf("missing metadata key %s", key)).Err()
				}
				ctx = context.WithValue(ctx, key, values)
			}
		}
		ctx = context.WithValue(ctx, constant.OperationID, md.Get(constant.OperationID)[0])
		if opts := md.Get(constant.OpUserID); len(opts) == 1 {
			ctx = context.WithValue(ctx, constant.OpUserID, opts[0])
		}
		if opts := md.Get(constant.OpUserPlatform); len(opts) == 1 {
			ctx = context.WithValue(ctx, constant.OpUserPlatform, opts[0])
		}
		if opts := md.Get(constant.ConnID); len(opts) == 1 {
			ctx = context.WithValue(ctx, constant.ConnID, opts[0])
		}
		return handler(ctx, req)
	})
}
