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
