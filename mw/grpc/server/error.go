package server

import (
	"context"
	"errors"

	"github.com/openimsdk/protocol/errinfo"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mw/specialerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func GrpcServerErrorConvert() grpc.ServerOption {
	type grpcError interface {
		error
		GRPCStatus() *status.Status
	}
	return grpc.ChainUnaryInterceptor(func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		resp, err = handler(ctx, req)
		if err == nil {
			return
		}
		var grpcErr grpcError
		if errors.As(err, &grpcErr) {
			return
		}
		err = codeErrorToGrpcError(ctx, getCodeError(err))
		return
	})
}

func getCodeError(err error) errs.CodeError {
    if codeErr := specialerror.ErrCode(err); codeErr != nil {
        return codeErr
    }
    return errs.ErrInternalServer.WithDetail(errs.Unwrap(err).Error())
}

func codeErrorToGrpcError(ctx context.Context, codeErr errs.CodeError) error {
	grpcStatus := status.New(codes.Code(codeErr.Code()), codeErr.Msg())
	if detail := codeErr.Detail(); detail != "" {
		errInfo := &errinfo.ErrorInfo{Cause: detail}
		details, err := grpcStatus.WithDetails(errInfo)
		if err == nil {
			return details.Err()
		} else {
			log.ZError(ctx, "rpc server response WithDetails failed", err, "codeErr", codeErr)
		}
	}
	return grpcStatus.Err()
}
