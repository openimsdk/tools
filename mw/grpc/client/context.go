package client

import (
	"context"

	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/tools/errs"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

func GrpcClientContext() grpc.DialOption {
	return grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md := metadata.Pairs()
		if keys, _ := ctx.Value(constant.RpcCustomHeader).([]string); len(keys) > 0 {
			for _, key := range keys {
				val, ok := ctx.Value(key).([]string)
				if !ok {
					return errs.ErrInternalServer.WrapMsg("ctx missing key", "key", key)
				}
				if len(val) == 0 {
					return errs.ErrInternalServer.WrapMsg("ctx key value is empty", "key", key)
				}
				md.Set(key, val...)
			}
			md.Set(constant.RpcCustomHeader, keys...)
		}
		operationID, ok := ctx.Value(constant.OperationID).(string)
		if !ok {
			return errs.ErrArgs.WrapMsg("ctx missing operationID")
		}
		md.Set(constant.OperationID, operationID)
		opUserID, ok := ctx.Value(constant.OpUserID).(string)
		if ok {
			md.Set(constant.OpUserID, opUserID)
			// checkArgs = append(checkArgs, constant.OpUserID, opUserID)
		}
		opUserIDPlatformID, ok := ctx.Value(constant.OpUserPlatform).(string)
		if ok {
			md.Set(constant.OpUserPlatform, opUserIDPlatformID)
		}
		connID, ok := ctx.Value(constant.ConnID).(string)
		if ok {
			md.Set(constant.ConnID, connID)
		}
		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	})
}
