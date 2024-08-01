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
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	ginApiResponseKey = "gin_api_response_key"
)

func ginJson(c *gin.Context, resp *ApiResponse) {
	c.Set(ginApiResponseKey, resp)
	c.JSON(http.StatusOK, resp)
}

func GetGinApiResponse(c *gin.Context) *ApiResponse {
	val, ok := c.Get(ginApiResponseKey)
	if !ok {
		return nil
	}
	resp, _ := val.(*ApiResponse)
	return resp
}

func GinError(c *gin.Context, err error) {
	ginJson(c, ParseError(err))
}

func GinSuccess(c *gin.Context, data any) {
	ginJson(c, ApiSuccess(data))
}
