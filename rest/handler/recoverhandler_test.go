package handler

import (
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithPanic(t *testing.T) {
	handler := RecoverHandler(func(ctx *fasthttp.RequestCtx) {
		panic("whatever")
	})
	ln := fasthttputil.NewInmemoryListener()
	s := fasthttp.Server{
		Handler: handler,
	}
	go s.Serve(ln) //nolint:errcheck
	c := &fasthttp.HostClient{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://localhost")
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
}

func TestWithoutPanic(t *testing.T) {
	handler := RecoverHandler(func(ctx *fasthttp.RequestCtx) {
	})
	ln := fasthttputil.NewInmemoryListener()
	s := fasthttp.Server{
		Handler: handler,
	}
	go s.Serve(ln) //nolint:errcheck
	c := &fasthttp.HostClient{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://localhost")
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}
