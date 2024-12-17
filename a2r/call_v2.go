package a2r

import (
	"context"

	"github.com/gin-gonic/gin"
	"github.com/openimsdk/tools/apiresp"
	"github.com/openimsdk/tools/checker"
	"google.golang.org/grpc"
)

func CallV2[A, B any](c *gin.Context, rpc func(ctx context.Context, req *A, options ...grpc.CallOption) (*B, error), opts ...*Option[A, B]) {
	req, err := ParseRequestNotCheck[A](c)
	if err != nil {
		apiresp.GinError(c, err)
		return
	}
	for _, opt := range opts {
		if opt.BindAfter == nil {
			continue
		}
		if err := opt.BindAfter(req); err != nil {
			apiresp.GinError(c, err) // args option error
			return
		}
	}
	if err := checker.Validate(req); err != nil {
		apiresp.GinError(c, err) // args option error
		return
	}
	resp, err := rpc(c, req)
	if err != nil {
		apiresp.GinError(c, err) // rpc call failed
		return
	}
	for _, opt := range opts {
		if opt.RespAfter == nil {
			continue
		}
		if err := opt.RespAfter(resp); err != nil {
			apiresp.GinError(c, err) // resp option error
			return
		}
	}
	apiresp.GinSuccess(c, resp) // rpc call success
}
