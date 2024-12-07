package handler

import (
	"bytes"
	"errors"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/r27153733/fastgozero/rest/internal"
	"github.com/stretchr/testify/assert"
)

func TestLogHandler(t *testing.T) {
	handlers := []func(handler fasthttp.RequestHandler) fasthttp.RequestHandler{
		LogHandler,
		DetailedLogHandler,
	}

	for _, logHandler := range handlers {
		handler := logHandler(func(ctx *fasthttp.RequestCtx) {
			internal.LogCollectorFromContext(ctx).Append("anything")
			ctx.Response.Header.Set("X-Test", "test")
			ctx.SetStatusCode(http.StatusServiceUnavailable)
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
		err := c.Do(req, resp)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode())
		assert.Equal(t, "test", string(resp.Header.Peek("X-Test")))
		assert.Equal(t, "content", string(resp.Body()))
	}
}

func TestLogHandlerVeryLong(t *testing.T) {
	var buf bytes.Buffer
	for i := 0; i < limitBodyBytes<<1; i++ {
		buf.WriteByte('a')
	}

	handler := LogHandler(func(ctx *fasthttp.RequestCtx) {
		internal.LogCollectorFromContext(ctx).Append("anything")
		ctx.Request.Body()
		ctx.Response.Header.Set("X-Test", "test")
		ctx.SetStatusCode(http.StatusServiceUnavailable)
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
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("http://localhost")
	req.SetBodyStream(&buf, -1)
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode())
	assert.Equal(t, "test", string(resp.Header.Peek("X-Test")))
	assert.Equal(t, "content", string(resp.Body()))
}

func TestLogHandlerSlow(t *testing.T) {
	handlers := []func(handler fasthttp.RequestHandler) fasthttp.RequestHandler{
		LogHandler,
		DetailedLogHandler,
	}

	for _, logHandler := range handlers {
		handler := logHandler(func(ctx *fasthttp.RequestCtx) {
			time.Sleep(defaultSlowThreshold + time.Millisecond*50)
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
}

//	func TestDetailedLogHandler_Hijack(t *testing.T) {
//		resp := httptest.NewRecorder()
//		writer := &detailLoggedResponseWriter{
//			writer: response.NewWithCodeResponseWriter(resp),
//		}
//		assert.NotPanics(t, func() {
//			_, _, _ = writer.Hijack()
//		})
//
//		writer = &detailLoggedResponseWriter{
//			writer: response.NewWithCodeResponseWriter(resp),
//		}
//		assert.NotPanics(t, func() {
//			_, _, _ = writer.Hijack()
//		})
//
//		writer = &detailLoggedResponseWriter{
//			writer: response.NewWithCodeResponseWriter(mockedHijackable{
//				ResponseRecorder: resp,
//			}),
//		}
//		assert.NotPanics(t, func() {
//			_, _, _ = writer.Hijack()
//		})
//	}
func TestSetSlowThreshold(t *testing.T) {
	assert.Equal(t, defaultSlowThreshold, slowThreshold.Load())
	SetSlowThreshold(time.Second)
	assert.Equal(t, time.Second, slowThreshold.Load())
}

func TestWrapMethodWithColor(t *testing.T) {
	// no tty
	assert.Equal(t, http.MethodGet, wrapMethod(http.MethodGet))
	assert.Equal(t, http.MethodPost, wrapMethod(http.MethodPost))
	assert.Equal(t, http.MethodPut, wrapMethod(http.MethodPut))
	assert.Equal(t, http.MethodDelete, wrapMethod(http.MethodDelete))
	assert.Equal(t, http.MethodPatch, wrapMethod(http.MethodPatch))
	assert.Equal(t, http.MethodHead, wrapMethod(http.MethodHead))
	assert.Equal(t, http.MethodOptions, wrapMethod(http.MethodOptions))
	assert.Equal(t, http.MethodConnect, wrapMethod(http.MethodConnect))
	assert.Equal(t, http.MethodTrace, wrapMethod(http.MethodTrace))
}

func TestWrapStatusCodeWithColor(t *testing.T) {
	// no tty
	assert.Equal(t, "200", wrapStatusCode(http.StatusOK))
	assert.Equal(t, "302", wrapStatusCode(http.StatusFound))
	assert.Equal(t, "404", wrapStatusCode(http.StatusNotFound))
	assert.Equal(t, "500", wrapStatusCode(http.StatusInternalServerError))
	assert.Equal(t, "503", wrapStatusCode(http.StatusServiceUnavailable))
}

//func TestDumpRequest(t *testing.T) {
//	const errMsg = "error"
//	r := httptest.NewRequest(http.MethodGet, "http://localhost", http.NoBody)
//	r.Body = mockedReadCloser{errMsg: errMsg}
//	assert.Equal(t, errMsg, dumpRequest(r))
//}

func BenchmarkLogHandler(b *testing.B) {
	b.ReportAllocs()

	handler := LogHandler(func(ctx *fasthttp.RequestCtx) {
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
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://localhost")

	for i := 0; i < b.N; i++ {
		resp := fasthttp.AcquireResponse()
		err := c.Do(req, resp)
		if err != nil {
			b.Fatal(err)
		}
		fasthttp.ReleaseResponse(resp)
	}
}

type mockedReadCloser struct {
	errMsg string
}

func (m mockedReadCloser) Read(_ []byte) (n int, err error) {
	return 0, errors.New(m.errMsg)
}

func (m mockedReadCloser) Close() error {
	return nil
}
