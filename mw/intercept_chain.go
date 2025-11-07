package mw

import (
	"context"

	"google.golang.org/grpc"
)

func InterceptChain(intercepts ...grpc.UnaryServerInterceptor) grpc.UnaryServerInterceptor {
	l := len(intercepts)
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		chain := func(currentInter grpc.UnaryServerInterceptor, currentHandler grpc.UnaryHandler) grpc.UnaryHandler {
			return func(currentCtx context.Context, currentReq any) (any, error) {
				return currentInter(
					currentCtx,
					currentReq,
					info,
					currentHandler)
			}
		}
		chainHandler := handler
		for i := l - 1; i >= 0; i-- {
			chainHandler = chain(intercepts[i], chainHandler)
		}
		return chainHandler(ctx, req)
	}
}
