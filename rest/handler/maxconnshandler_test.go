package handler

import (
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"net"
	"net/http"
	"sync"
	"testing"

	"github.com/r27153733/fastgozero/core/lang"
	"github.com/stretchr/testify/assert"
)

const conns = 4

func TestMaxConnsHandler(t *testing.T) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(conns)
	done := make(chan lang.PlaceholderType)
	defer close(done)

	maxConns := MaxConnsHandler(conns)
	handler := maxConns(func(ctx *fasthttp.RequestCtx) {
		waitGroup.Done()
		<-done
	})
	ln := fasthttputil.NewInmemoryListener()
	s := fasthttp.Server{
		Handler: handler,
	}
	go s.Serve(ln) //nolint:errcheck
	for i := 0; i < conns; i++ {
		go func() {
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
				t.Error(err)
				return
			}
		}()
	}

	waitGroup.Wait()
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

func TestWithoutMaxConnsHandler(t *testing.T) {
	const (
		key   = "block"
		value = "1"
	)
	var waitGroup sync.WaitGroup
	waitGroup.Add(conns)
	done := make(chan lang.PlaceholderType)
	defer close(done)

	maxConns := MaxConnsHandler(0)
	handler := maxConns(func(ctx *fasthttp.RequestCtx) {
		val := ctx.Request.Header.Peek(key)
		if string(val) == value {
			waitGroup.Done()
			<-done
		}
	})
	ln := fasthttputil.NewInmemoryListener()
	s := fasthttp.Server{
		Handler: handler,
	}
	go s.Serve(ln) //nolint:errcheck

	for i := 0; i < conns; i++ {
		go func() {
			c := &fasthttp.HostClient{
				Dial: func(addr string) (net.Conn, error) {
					return ln.Dial()
				},
			}
			req := fasthttp.AcquireRequest()
			resp := fasthttp.AcquireResponse()
			req.Header.SetMethod(fasthttp.MethodGet)
			req.SetRequestURI("http://localhost")
			req.Header.Set(key, value)
			err := c.Do(req, resp)
			if err != nil {
				t.Error(err)
				return
			}
		}()
	}

	waitGroup.Wait()
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
		t.Error(err)
		return
	}
	assert.Equal(t, http.StatusOK, resp.StatusCode())
}
