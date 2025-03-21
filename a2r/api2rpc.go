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

package a2r

import (
	"context"
	"github.com/openimsdk/tools/checker"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/openimsdk/tools/apiresp"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/utils/jsonutil"
	"google.golang.org/grpc"
)

type Option[A, B any] struct {
	// BindAfter is called after the req is bind from ctx.
	BindAfter func(*A) error
	// RespAfter is called after the resp is return from rpc.
	RespAfter func(*B) error
}

func Call[A, B, C any](c *gin.Context, rpc func(client C, ctx context.Context, req *A, options ...grpc.CallOption) (*B, error), client C, opts ...*Option[A, B]) {
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
	resp, err := rpc(client, c, req)
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

func ParseRequestNotCheck[T any](c *gin.Context) (*T, error) {
	var req T
	if err := c.ShouldBindWith(&req, jsonBind); err != nil {
		return nil, errs.NewCodeError(errs.ArgsError, err.Error())
	}
	return &req, nil
}

func ParseRequest[T any](c *gin.Context) (*T, error) {
	req, err := ParseRequestNotCheck[T](c)
	if err != nil {
		return nil, err
	}
	if err := checker.Validate(&req); err != nil {
		return nil, err
	}
	return req, nil
}

type jsonBinding struct{}

var jsonBind binding.Binding = jsonBinding{}

func (jsonBinding) Name() string {
	return "json"
}

func (b jsonBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errs.New("invalid request").Wrap()
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return errs.WrapMsg(err, "read request body failed", "method", req.Method, "url", req.URL.String())
	}
	return errs.Wrap(b.BindBody(body, obj))
}

func (jsonBinding) BindBody(body []byte, obj any) error {
	if err := jsonutil.JsonUnmarshal(body, obj); err != nil {
		return err
	}
	if binding.Validator == nil {
		return nil
	}
	return errs.Wrap(binding.Validator.ValidateStruct(obj))
}
