package handler

import (
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"github.com/zeromicro/go-zero/core/codec"
	"github.com/zeromicro/go-zero/rest/httpx"
)

func TestGunzipHandler(t *testing.T) {
	const message = "hello world"
	var wg sync.WaitGroup
	wg.Add(1)
	handler := GunzipHandler(func(ctx *fasthttp.RequestCtx) {
		body := ctx.PostBody()
		assert.Equal(t, string(body), message)
		wg.Done()
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
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetBody(codec.Gzip([]byte(message)))
	req.SetRequestURI("http://localhost")
	req.Header.Set(httpx.ContentEncoding, gzipEncoding)
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	wg.Wait()
}

func TestGunzipHandler_NoGzip(t *testing.T) {
	const message = "hello world"
	var wg sync.WaitGroup
	wg.Add(1)
	handler := GunzipHandler(func(ctx *fasthttp.RequestCtx) {
		body := ctx.PostBody()
		assert.Equal(t, string(body), message)
		wg.Done()
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
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetBody([]byte(message))
	req.SetRequestURI("http://localhost")
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusOK, resp.StatusCode())
	wg.Wait()
}

func TestGunzipHandler_NoGzipButTelling(t *testing.T) {
	const message = "hello world"
	handler := GunzipHandler(func(ctx *fasthttp.RequestCtx) {})
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
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetBody([]byte(message))
	req.SetRequestURI("http://localhost")
	req.Header.Set(httpx.ContentEncoding, gzipEncoding)
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
}
