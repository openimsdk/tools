package api

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/openimsdk/protocol/constant"
	"github.com/openimsdk/tools/apiresp"
	"github.com/openimsdk/tools/errs"
	"github.com/openimsdk/tools/log"
)

func GinLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		httpID := uuid.New().String()
		c.Writer.Header().Set("X-HTTP-ID", httpID)
		operationID := c.Request.Header.Get(constant.OperationID)
		log.ZDebug(c, "http request received", "httpID", httpID,
			"operationID", operationID,
			"httpProto", c.Request.Proto, "remoteAddr", c.Request.RemoteAddr,
			"forwardedFor", c.Request.Header.Get("X-Forwarded-For"),
			"contentEncoding", c.Request.Header.Get("Content-Encoding"),
			"contentType", c.Request.Header.Get("Content-Type"),
			"contentLength", c.Request.ContentLength,
			"method", c.Request.Method, "host", c.Request.Host,
			"requestURI", c.Request.RequestURI)
		body, err := io.ReadAll(io.LimitReader(c.Request.Body, 1024*1024*16))
		if err != nil {
			log.ZWarn(c, "read request body failed", err, "httpID", httpID, "operationID", operationID, "method", c.Request.Method, "requestURI", c.Request.RequestURI, "cost", time.Since(start))
			c.Abort()
			apiresp.GinError(c, errs.ErrArgs.WrapMsg("read http request body failed"))
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		log.ZInfo(c, "http request body", "httpID", httpID, "operationID", operationID, "method", c.Request.Method, "requestURI", c.Request.RequestURI, "reqBody", string(body), "cost", time.Since(start))
		response := &httpResponse{
			ResponseWriter: c.Writer,
		}
		c.Writer = response
		c.Next()
		location := response.Header().Get("Location")
		if location == "" {
			log.ZInfo(c, "http request processed", "httpID", httpID, "operationID", operationID, "method", c.Request.Method, "requestURI", c.Request.RequestURI, "status", c.Writer.Status(), "cost", time.Since(start), "reqBody", string(body), "respBody", response.buf.String())
		} else {
			log.ZInfo(c, "http request processed", "httpID", httpID, "operationID", operationID, "method", c.Request.Method, "requestURI", c.Request.RequestURI, "status", c.Writer.Status(), "cost", time.Since(start), "reqBody", string(body), "respBody", response.buf.String(), "location", location)
		}
	}
}

type httpResponse struct {
	gin.ResponseWriter
	buf bytes.Buffer
}

func (r *httpResponse) Write(b []byte) (int, error) {
	r.buf.Write(b)
	return r.ResponseWriter.Write(b)
}
