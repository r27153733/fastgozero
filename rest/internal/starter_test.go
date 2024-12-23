package internal

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/r27153733/fastgozero/core/proc"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func NotFoundHandler(ctx *fasthttp.RequestCtx) {
	ctx.NotFound()
}

func TestStartHttp(t *testing.T) {
	svr := httptest.NewUnstartedServer(http.NotFoundHandler())
	fields := strings.Split(svr.Listener.Addr().String(), ":")
	port, err := strconv.Atoi(fields[1])
	assert.Nil(t, err)
	err = StartHttp(fields[0], port, NotFoundHandler, func(svr *fasthttp.Server) {
		svr.IdleTimeout = 0
	})
	assert.NotNil(t, err)
	proc.WrapUp()
}

func TestStartHttps(t *testing.T) {
	svr := httptest.NewTLSServer(http.NotFoundHandler())
	fields := strings.Split(svr.Listener.Addr().String(), ":")
	port, err := strconv.Atoi(fields[1])
	assert.Nil(t, err)
	err = StartHttps(fields[0], port, "", "", NotFoundHandler, func(svr *fasthttp.Server) {
		svr.IdleTimeout = 0
	})
	assert.NotNil(t, err)
	proc.WrapUp()
}
