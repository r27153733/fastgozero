package internal

import (
	"github.com/valyala/fasthttp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildHeadersNoValue(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)

	req.Request.Header.Add("a", "b")
	assert.Nil(t, ProcessHeaders(&req.Request.Header))
}

func TestBuildHeadersWithValues(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)

	req.Request.Header.Add("grpc-metadata-a", "b")
	req.Request.Header.Add("grpc-metadata-b", "b")
	assert.ElementsMatch(t, []string{"gateway-A:b", "gateway-B:b"}, ProcessHeaders(&req.Request.Header))
}
