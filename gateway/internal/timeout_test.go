package internal

import (
	"github.com/valyala/fasthttp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetTimeout(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)

	req.Request.Header.Set(grpcTimeoutHeader, "1s")
	timeout := GetTimeout(&req.Request.Header, time.Second*5)
	assert.Equal(t, time.Second, timeout)
}

func TestGetTimeoutDefault(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)

	timeout := GetTimeout(&req.Request.Header, time.Second*5)
	assert.Equal(t, time.Second*5, timeout)
}
