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
	"fmt"
	"math"

	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/protocol/errinfo"
	"github.com/openimsdk/tools/checker"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mw/specialerror"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func rpcString(v any) string {
	if s, ok := v.(interface{ String() string }); ok {
		return s.String()
	}
	return fmt.Sprintf("%+v", v)
}

func RpcServerInterceptor(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
	funcName := info.FullMethod
	logRequest(ctx, funcName, req)
	if err := validateMetadata(ctx); err != nil {
		return nil, err
	}
	ctx, err := enrichContextWithMetadata(ctx)
	if err != nil {
		return nil, err
	}
	resp, err := processRequest(ctx, req, handler, funcName)
	if err != nil {
		logErrorResponse(ctx, funcName, req, err)
		return nil, prepareErrorDetail(err)
	}
	log.ZInfo(ctx, "rpc server resp", "funcName", funcName, "resp", rpcString(resp))
	return resp, nil
}

func validateMetadata(ctx context.Context) error {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return status.New(codes.InvalidArgument, "missing metadata").Err()
	}
	if len(md.Get(constant.OperationID)) != 1 {
		return status.New(codes.InvalidArgument, "operationID error").Err()
	}
	return nil
}

func enrichContextWithMetadata(ctx context.Context) (context.Context, error) {
	md, _ := metadata.FromIncomingContext(ctx) // Already validated
	for _, key := range md.Get(constant.RpcCustomHeader) {
		values := md.Get(key)
		if len(values) == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "missing metadata key %s", key)
		}
		ctx = context.WithValue(ctx, key, values[0]) // Storing only the first value for simplicity
	}
	return ctx, nil
}

func processRequest(ctx context.Context, req any, handler grpc.UnaryHandler, funcName string) (resp any, err error) {
	if err := checker.Validate(req); err != nil {
		return nil, err
	}
	return handler(ctx, req)
}

func logRequest(ctx context.Context, funcName string, req any) {
	log.ZInfo(ctx, "rpc server req", "funcName", funcName, "req", rpcString(req))
}

func logErrorResponse(ctx context.Context, funcName string, req any, err error) {
	unwrap := errs.Unwrap(err)
	codeErr := specialerror.ErrCode(unwrap)
	if codeErr == nil {
		log.ZError(ctx, "rpc InternalServer error", err, "req", req)
		codeErr = errs.ErrInternalServer
	}
	code := codeErr.Code()
	if code <= 0 || int64(code) > int64(math.MaxUint32) {
		log.ZError(ctx, "rpc UnknownError", err, "rpc UnknownCode:", int64(code))
		// code = errs.ServerInternalError
	}
	log.ZError(ctx, "rpc server resp", err, "funcName", funcName)
}

func prepareErrorDetail(err error) error {
	grpcStatus := status.New(codes.Internal, err.Error())
	errInfo := &errinfo.ErrorInfo{Cause: err.Error()}
	details, err := grpcStatus.WithDetails(errInfo)
	if err != nil {
		log.ZWarn(context.Background(), "rpc server resp WithDetails error", err)
		return errs.WrapMsg(err, "rpc server resp WithDetails error", "err", err)
	}
	return details.Err()
}

func GrpcServer() grpc.ServerOption {
	return grpc.ChainUnaryInterceptor(RpcServerInterceptor)
}
