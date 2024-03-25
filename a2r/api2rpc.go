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
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin/binding"
	"github.com/openimsdk/tools/utils/jsonutil"

	"github.com/gin-gonic/gin"
	"github.com/openimsdk/tools/apiresp"
	"github.com/openimsdk/tools/checker"
	"github.com/openimsdk/tools/errs"
	"google.golang.org/grpc"
)

func Call[A, B, C any](rpc func(client C, ctx context.Context, req *A, options ...grpc.CallOption) (*B, error), client C, c *gin.Context) {
	var req A
	if err := c.ShouldBindWith(&req, jsonBind); err != nil {
		apiresp.GinError(c, errs.ErrArgs.WithDetail(err.Error()).Wrap()) // args error
		return
	}
	if err := checker.Validate(&req); err != nil {
		apiresp.GinError(c, err) // args validate error
		return
	}
	data, err := rpc(client, c, &req)
	if err != nil {
		apiresp.GinError(c, err) // rpc call failed
		return
	}
	apiresp.GinSuccess(c, data) // rpc call success
}

var jsonBind binding.Binding = jsonBinding{}

type jsonBinding struct{}

func (jsonBinding) Name() string {
	return "json"
}

func (b jsonBinding) Bind(req *http.Request, obj any) error {
	if req == nil || req.Body == nil {
		return errors.New("invalid request")
	}
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return err
	}
	return b.BindBody(body, obj)
}

func (jsonBinding) BindBody(body []byte, obj any) error {
	if err := jsonutil.JsonUnmarshal(body, obj); err != nil {
		return err
	}
	if binding.Validator == nil {
		return nil
	}
	return binding.Validator.ValidateStruct(obj)
}
