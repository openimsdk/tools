package mw

import (
	"context"
	"math"

	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
	"github.com/openimsdk/tools/mw/specialerror"
)

func HandleCallError(ctx context.Context, funcName string, args []any, err error) error {
	log.ZWarn(ctx, "fn call WithDetails Response is error", formatError(err), "funcName", funcName, "args", args)
	unwrap := errs.Unwrap(err)
	codeErr := specialerror.ErrCode(unwrap)
	if codeErr == nil {
		log.ZError(ctx, "internal server error", formatError(err), "funcName", funcName, "args", args)
		codeErr = errs.ErrInternalServer
	}
	code := codeErr.Code()
	if code <= 0 || int64(code) > int64(math.MaxUint32) {
		log.ZError(ctx, "unknown error code", formatError(err), "funcName", funcName, "args", args, "unknown code:", int64(code))
		code = errs.ServerInternalError
	}

	return nil
}
