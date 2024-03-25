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

package apiresp

import (
	"encoding/json"
	"reflect"

	"github.com/openimsdk/tools/utils/jsonutil"

	"github.com/openimsdk/tools/errs"
)

type ApiResponse struct {
	ErrCode int    `json:"errCode"`
	ErrMsg  string `json:"errMsg"`
	ErrDlt  string `json:"errDlt"`
	Data    any    `json:"data,omitempty"`
}

func (r *ApiResponse) MarshalJSON() ([]byte, error) {
	if r.Data != nil {
		data, err := jsonutil.JsonMarshal(r.Data)
		if err != nil {
			return nil, err
		}
		r.Data = json.RawMessage(data)
	}
	return jsonutil.JsonMarshal(r)
}

func isAllFieldsPrivate(v any) bool {
	typeOf := reflect.TypeOf(v)
	if typeOf == nil {
		return false
	}
	for typeOf.Kind() == reflect.Ptr {
		typeOf = typeOf.Elem()
	}
	if typeOf.Kind() != reflect.Struct {
		return false
	}
	num := typeOf.NumField()
	for i := 0; i < num; i++ {
		c := typeOf.Field(i).Name[0]
		if c >= 'A' && c <= 'Z' {
			return false
		}
	}
	return true
}

func ApiSuccess(data any) *ApiResponse {
	if format, ok := data.(ApiFormat); ok {
		format.ApiFormat()
	}
	if isAllFieldsPrivate(data) {
		return &ApiResponse{Data: json.RawMessage(nil)}
	}
	return &ApiResponse{Data: data}
}

func ParseError(err error) *ApiResponse {
	if err == nil {
		return ApiSuccess(nil)
	}
	unwrap := errs.Unwrap(err)
	if codeErr, ok := unwrap.(errs.CodeError); ok {
		resp := ApiResponse{ErrCode: codeErr.Code(), ErrMsg: codeErr.Msg(), ErrDlt: codeErr.Detail()}
		if resp.ErrDlt == "" {
			resp.ErrDlt = err.Error()
		}
		return &resp
	}
	return &ApiResponse{ErrCode: errs.ServerInternalError, ErrMsg: err.Error()}
}
