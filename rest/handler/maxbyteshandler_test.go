package handler

import (
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMaxBytesHandler(t *testing.T) {
	maxb := MaxBytesHandler(10)
	handler := maxb(func(ctx *fasthttp.RequestCtx) {})
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
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetRequestURI("http://localhost")
	req.SetBodyString("123456789012345")
	resp := fasthttp.AcquireResponse()
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusRequestEntityTooLarge, resp.StatusCode())

	req.SetBodyString("12345")
	resp.Reset()
	err = c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}

func TestMaxBytesHandlerNoLimit(t *testing.T) {
	maxb := MaxBytesHandler(-1)
	handler := maxb(func(ctx *fasthttp.RequestCtx) {})
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
	req.Header.SetMethod(fasthttp.MethodPost)
	req.Header.SetRequestURI("http://localhost")
	req.SetBodyString("123456789012345")
	resp := fasthttp.AcquireResponse()
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}
