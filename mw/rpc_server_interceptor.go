package mw

import (
	"context"
	"fmt"
	"github.com/openimsdk/tools/checker"
	"math"

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
	method := info.FullMethod
	md, err := validateMetadata(ctx)
	if err != nil {
		return nil, err
	}
	ctx, err = enrichContextWithMetadata(ctx, md)
	if err != nil {
		return nil, err
	}
	defer func() {
		if r := recover(); r != nil {
			err = errs.ErrPanic(r)
			log.ZPanic(ctx, "rpc server panic", err, "method", method, "req", req)
		}
	}()
	log.ZInfo(ctx, "rpc server request", "method", method, "req", req)
	if err := checker.Validate(req); err != nil {
		return nil, handleError(ctx, method, req, err)
	}
	resp, err := handler(ctx, req)
	if err != nil {
		return nil, handleError(ctx, method, req, err)
	}
	log.ZInfo(ctx, "rpc server response success", "method", method, "req", req, "resp", resp)
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

func handleError(ctx context.Context, method string, req any, err error) error {
	codeErr := specialerror.ErrCode(errs.Unwrap(err))
	if codeErr == nil {
		codeErr = errs.ErrInternalServer
	}
	code := codeErr.Code()
	if code <= 0 || int64(code) > int64(math.MaxUint32) {
		code = errs.ServerInternalError
	}
	if _, ok := errs.Unwrap(err).(errs.CodeError); ok {
		log.ZAdaptive(ctx, "rpc server response failed", err, "method", method, "req", req)
	} else {
		log.ZAdaptive(ctx, "rpc server response failed", err, "rawerror", err, "method", method, "req", req)
	}
	grpcStatus := status.New(codes.Code(code), err.Error())
	errInfo := &errinfo.ErrorInfo{Cause: err.Error()}
	details, err := grpcStatus.WithDetails(errInfo)
	if err != nil {
		log.ZError(ctx, "rpc server response WithDetails failed", err, "method", method, "req", req)
		return errs.WrapMsg(err, "rpc server resp WithDetails error", "err", err)
	}
	return details.Err()
}

func GrpcServer() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(RpcServerInterceptor)
}
