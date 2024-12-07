package handler

import (
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"net"
	"net/http"
	"testing"

	"github.com/r27153733/fastgozero/core/load"
	"github.com/r27153733/fastgozero/core/stat"
	"github.com/stretchr/testify/assert"
)

func TestSheddingHandlerAccept(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	shedder := mockShedder{
		allow: true,
	}
	sheddingHandler := SheddingHandler(shedder, metrics)
	handler := sheddingHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set("X-Test", "test")
		ctx.Response.AppendBody([]byte("content"))
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
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, "test", string(resp.Header.Peek("X-Test")))
	assert.Equal(t, "content", string(resp.Body()))
}

func TestSheddingHandlerFail(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	shedder := mockShedder{
		allow: true,
	}
	sheddingHandler := SheddingHandler(shedder, metrics)
	handler := sheddingHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(http.StatusServiceUnavailable)
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
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode())
}

func TestSheddingHandlerReject(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	shedder := mockShedder{
		allow: false,
	}
	sheddingHandler := SheddingHandler(shedder, metrics)
	handler := sheddingHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(http.StatusOK)
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
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode())
}

func TestSheddingHandlerNoShedding(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	sheddingHandler := SheddingHandler(nil, metrics)
	handler := sheddingHandler(func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(http.StatusOK)
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

type mockShedder struct {
	allow bool
}

func (s mockShedder) Allow() (load.Promise, error) {
	if s.allow {
		return mockPromise{}, nil
	}

	return nil, load.ErrServiceOverloaded
}

type mockPromise struct{}

func (p mockPromise) Pass() {
}

func (p mockPromise) Fail() {
}
