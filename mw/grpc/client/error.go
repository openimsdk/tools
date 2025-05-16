package client

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/openimsdk/protocol/errinfo"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
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
		if sta.Code() == codes.Unavailable {
			target := cc.Target()
			if index := strings.LastIndex(target, "/"); index >= 0 {
				target = target[index+1:]
			}
			msg := fmt.Sprintf("grpc service %s down, grpc message %s", target, sta.Message())
			return errs.NewCodeError(errs.ServerInternalError, msg).Wrap()
		}
		if sta.Code() < 100 {
			return errs.ErrInternalServer.WrapMsg(err.Error())
		}
		if details := sta.Details(); len(details) > 0 {
			if errInfo, ok := details[0].(*errinfo.ErrorInfo); ok {
				detail := strings.Join(errInfo.Warp, "->") + errInfo.Cause
				return errs.NewCodeError(int(sta.Code()), sta.Message()).WithDetail(detail).Wrap()
			}
		}
		return errs.NewCodeError(int(sta.Code()), sta.Message()).Wrap()
	})
}
