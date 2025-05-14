package client

import (
	"context"
	"errors"
	"strings"

	"github.com/openimsdk/protocol/errinfo"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

func GrpcClientErrorConvert() grpc.DialOption {
	type grpcError interface {
		error
		GRPCStatus() *status.Status
	}
	return grpc.WithChainUnaryInterceptor(func(ctx context.Context, method string, req, reply any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		err := invoker(ctx, method, req, reply, cc, opts...)
		if err == nil {
			return nil
		}
		var codeErr errs.CodeError
		if errors.As(err, &codeErr) {
			return err
		}
		var grpcErr grpcError
		if !errors.As(err, &grpcErr) {
			log.ZError(ctx, "rpc client response failed not GRPCStatus", err, "method", method, "req", req)
			return errs.ErrInternalServer.WrapMsg(err.Error())
		}
		sta := grpcErr.GRPCStatus()
		if sta.Code() == 0 {
			log.ZError(ctx, "rpc client response failed GRPCStatus code is 0", err, "method", method, "req", req)
			return errs.NewCodeError(errs.ServerInternalError, err.Error()).Wrap()
		}
		if details := sta.Details(); len(details) > 0 {
			errInfo, ok := details[0].(*errinfo.ErrorInfo)
			if ok {
				s := strings.Join(errInfo.Warp, "->") + errInfo.Cause
				return errs.NewCodeError(int(sta.Code()), sta.Message()).WithDetail(s).Wrap()
			}
		}
		return errs.NewCodeError(int(sta.Code()), sta.Message()).Wrap()
	})
}
