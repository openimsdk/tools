package mw

import (
	"context"
	"fmt"
	"math"

	"github.com/openimsdk/tools/checker"

	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/errinfo"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mw/specialerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func RpcServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (_ any, err error) {
	funcName := info.FullMethod
	md, err := validateMetadata(ctx)
	if err != nil {
		return nil, err
	}
	ctx, err = enrichContextWithMetadata(ctx, md)
	if err != nil {
		return nil, err
	}
	log.ZInfo(ctx, fmt.Sprintf("RPC Server Request - %s", extractFunctionName(funcName)), "funcName", funcName, "req", req)
	if err := checker.Validate(req); err != nil {
		return nil, err
	}

	defer func() {
		if r := recover(); r != nil {
			err = errs.ErrPanic(r)
			log.ZPanic(ctx, "RpcServerInterceptor panic", err)
		}
	}()
	resp, err := handler(ctx, req)
	if err != nil {
		return nil, handleError(ctx, funcName, req, err)
	}
	log.ZInfo(ctx, fmt.Sprintf("RPC Server Response Success - %s", extractFunctionName(funcName)), "funcName", funcName, "resp", resp)
	return resp, nil
}

func validateMetadata(ctx context.Context) (metadata.MD, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.New(codes.InvalidArgument, "missing metadata").Err()
	}
	if len(md.Get(constant.OperationID)) != 1 {
		return nil, status.New(codes.InvalidArgument, "operationID error").Err()
	}
	return md, nil
}

func enrichContextWithMetadata(ctx context.Context, md metadata.MD) (context.Context, error) {
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
	return ctx, nil
}

func handleError(ctx context.Context, funcName string, req any, err error) error {
	log.ZWarn(ctx, "rpc server resp WithDetails error", err, "funcName", funcName)
	unwrap := errs.Unwrap(err)
	codeErr := specialerror.ErrCode(unwrap)
	if codeErr == nil {
		log.ZError(ctx, "rpc InternalServer error", err, "funcName", funcName, "req", req)
		codeErr = errs.ErrInternalServer
	}
	code := codeErr.Code()
	if code <= 0 || int64(code) > int64(math.MaxUint32) {
		log.ZError(ctx, "rpc UnknownError", err, "funcName", funcName, "rpc UnknownCode:", int64(code))
		code = errs.ServerInternalError
	}
	grpcStatus := status.New(codes.Code(code), err.Error())
	errInfo := &errinfo.ErrorInfo{Cause: err.Error()}
	details, err := grpcStatus.WithDetails(errInfo)
	if err != nil {
		log.ZWarn(ctx, "rpc server resp WithDetails error", err, "funcName", funcName)
		return errs.WrapMsg(err, "rpc server resp WithDetails error", "err", err)
	}
	log.ZWarn(ctx, fmt.Sprintf("RPC Server Response Error - %s", extractFunctionName(funcName)), details.Err(), "funcName", funcName, "req", req, "err", err)
	return details.Err()
}

func GrpcServer() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(RpcServerInterceptor)
}
