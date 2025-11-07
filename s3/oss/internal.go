package oss

import (
	"net/http"
	"net/url"
	_ "unsafe"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

//go:linkname signHeader github.com/aliyun/aliyun-oss-go-sdk/oss.Conn.signHeader
func signHeader(c oss.Conn, req *http.Request, canonicalizedResource string, credentials oss.Credentials)

//go:linkname getURLParams github.com/aliyun/aliyun-oss-go-sdk/oss.Conn.getURLParams
func getURLParams(c oss.Conn, params map[string]any) string

//go:linkname getURL github.com/aliyun/aliyun-oss-go-sdk/oss.urlMaker.getURL
func getURL(um urlMaker, bucket, object, params string) *url.URL

type urlMaker struct {
	Scheme  string
	NetLoc  string
	Type    int
	IsProxy bool
}
