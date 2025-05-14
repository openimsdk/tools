package server

import (
	"context"
	"errors"
	"math"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/mw/specialerror"
	"google.golang.org/grpc"
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
		err = getErrData(err)
		return
	})
}

func getErrData(err error) errs.CodeError {
	var (
		code        int
		msg, detail string
	)
	codeErr := specialerror.ErrCode(err)
	if codeErr != nil {
		code = codeErr.Code()
		msg = codeErr.Msg()
		detail = codeErr.Detail()
	} else {
		code = errs.ServerInternalError
	}
	if code <= 0 || int64(code) > int64(math.MaxUint32) {
		code = errs.ServerInternalError
	}

	if msg == "" || detail == "" {
		stringErr := specialerror.ErrString(err)
		wrapErr := specialerror.ErrWrapper(err)

		if stringErr != nil {
			if msg == "" {
				msg = stringErr.Error()
			}
		}

		if wrapErr != nil {
			if msg == "" {
				msg = wrapErr.Error()
			}
			if detail == "" {
				detail = wrapErr.Error()
			}
		}
	}
	if msg == "" {
		msg = err.Error()
	}
	if detail == "" {
		detail = msg
	}

	return errs.NewCodeError(code, msg).WithDetail(detail)
}
