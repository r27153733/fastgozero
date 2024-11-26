package handler

import (
	"fmt"
	"net"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"github.com/zeromicro/go-zero/core/stat"
)

func init() {
	stat.SetReporter(nil)
}

func TestBreakerHandlerAccept(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	breakerHandler := BreakerHandler(fasthttp.MethodGet, "/", metrics)
	handler := breakerHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("X-Test", "test")
		ctx.Response.AppendBodyString("content")
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
	req.Header.Set("X-Test", "test")
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	assert.Equal(t, "test", string(resp.Header.Peek("X-Test")))
	assert.Equal(t, "content", string(resp.Body()))
}

func TestBreakerHandlerFail(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	breakerHandler := BreakerHandler(fasthttp.MethodGet, "/", metrics)
	handler := breakerHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.SetStatusCode(fasthttp.StatusBadGateway)
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
	assert.Equal(t, fasthttp.StatusBadGateway, resp.StatusCode())
}

func TestBreakerHandler_4XX(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	breakerHandler := BreakerHandler(fasthttp.MethodGet, "/", metrics)
	handler := breakerHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
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

	for i := 0; i < 1000; i++ {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		req.Header.SetMethod(fasthttp.MethodGet)
		req.SetRequestURI("http://localhost")
		err := c.Do(req, resp)
		if err != nil {
			t.Fatal(err)
		}
	}

	const tries = 100
	var pass int
	for i := 0; i < tries; i++ {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		req.Header.SetMethod(fasthttp.MethodGet)
		req.SetRequestURI("http://localhost")
		err := c.Do(req, resp)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode() == http.StatusBadRequest {
			pass++
		}
	}

	assert.Equal(t, tries, pass)
}

func TestBreakerHandlerReject(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	breakerHandler := BreakerHandler(fasthttp.MethodGet, "/", metrics)
	handler := breakerHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.SetStatusCode(fasthttp.StatusInternalServerError)
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
	for i := 0; i < 1000; i++ {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		req.Header.SetMethod(fasthttp.MethodGet)
		req.SetRequestURI("http://localhost")
		err := c.Do(req, resp)
		if err != nil {
			t.Fatal(err)
		}
	}

	var drops int
	for i := 0; i < 100; i++ {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		req.Header.SetMethod(fasthttp.MethodGet)
		req.SetRequestURI("http://localhost")
		err := c.Do(req, resp)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode() == fasthttp.StatusServiceUnavailable {
			drops++
		}
	}

	assert.True(t, drops >= 80, fmt.Sprintf("expected to be greater than 80, but got %d", drops))
}
