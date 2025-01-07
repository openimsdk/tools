// Copyright © 2023 OpenIM. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package mw

import (
	"context"
	"strings"

	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/errinfo"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func GrpcClient() grpc.DialOption {
	return grpc.WithChainUnaryInterceptor(RpcClientInterceptor)
}

func RpcClientInterceptor(ctx context.Context, method string, req, resp any, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) (err error) {
	if ctx == nil {
		return errs.ErrInternalServer.WrapMsg("call rpc request context is nil")
	}
	ctx, err = getRpcContext(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			err = errs.ErrPanic(r)
			log.ZPanic(ctx, "rpc client panic", err, "method", method, "req", req)
		}
	}()
	log.ZInfo(ctx, "rpc client request", "method", method, "target", cc.Target(), "req", req)
	err = invoker(ctx, method, req, resp, cc, opts...)
	if err == nil {
		log.ZInfo(ctx, "rpc client response success", "method", method, "resp", resp)
		return nil
	}
	rpcErr, ok := err.(interface{ GRPCStatus() *status.Status })
	if !ok {
		log.ZError(ctx, "rpc client response failed not GRPCStatus", err, "method", method, "req", req)
		return errs.ErrInternalServer.WrapMsg(err.Error())
	}
	sta := rpcErr.GRPCStatus()
	if sta.Code() == 0 {
		log.ZError(ctx, "rpc client response failed GRPCStatus code is 0", err, "method", method, "req", req)
		return errs.NewCodeError(errs.ServerInternalError, err.Error()).Wrap()
	}
	if details := sta.Details(); len(details) > 0 {
		errInfo, ok := details[0].(*errinfo.ErrorInfo)
		if ok {
			s := strings.Join(errInfo.Warp, "->") + errInfo.Cause
			cErr := errs.NewCodeError(int(sta.Code()), sta.Message()).WithDetail(s).Wrap()
			log.ZAdaptive(ctx, "rpc client response failed", cErr, "method", method, "req", req)
			return cErr
		}
	}
	cErr := errs.NewCodeError(int(sta.Code()), sta.Message()).Wrap()
	log.ZAdaptive(ctx, "rpc client response failed", cErr, "method", method, "req", req)
	return cErr
}

func getRpcContext(ctx context.Context) (context.Context, error) {
	md := metadata.Pairs()
	if keys, _ := ctx.Value(constant.RpcCustomHeader).([]string); len(keys) > 0 {
		for _, key := range keys {
			val, ok := ctx.Value(key).([]string)
			if !ok {
				return nil, errs.ErrInternalServer.WrapMsg("ctx missing key", "key", key)
			}
			if len(val) == 0 {
				return nil, errs.ErrInternalServer.WrapMsg("ctx key value is empty", "key", key)
			}
			md.Set(key, val...)
		}
		md.Set(constant.RpcCustomHeader, keys...)
	}
	operationID, ok := ctx.Value(constant.OperationID).(string)
	if !ok {
		return nil, errs.ErrArgs.WrapMsg("ctx missing operationID")
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
	return metadata.NewOutgoingContext(ctx, md), nil
}

func extractFunctionName(funcName string) string {
	parts := strings.Split(funcName, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return ""
}
