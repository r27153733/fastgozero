package httpx

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestGetRemoteAddr(t *testing.T) {
	host := "8.8.8.8"
	r := new(fasthttp.RequestCtx)
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	r.Request.SetRequestURI("/")

	r.Request.Header.Set(xForwardedFor, host)
	assert.Equal(t, host, GetRemoteAddr(r))
}

//func TestGetRemoteAddrNoHeader(t *testing.T) {
//	r := new(fasthttp.RequestCtx)
//	r.Request.Header.SetMethod(fasthttp.MethodGet)
//	r.Request.SetRequestURI("/")
//
//	assert.True(t, len(GetRemoteAddr(r)) == 0)
//}
