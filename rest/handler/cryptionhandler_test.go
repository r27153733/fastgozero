package handler

import (
	"crypto/rand"
	"encoding/base64"
	"io"
	"net"
	"net/http"
	"testing"
	"testing/iotest"

	"github.com/r27153733/fastgozero/core/codec"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

const (
	reqText  = "ping"
	respText = "pong"
)

var aesKey = []byte(`PdSgVkYp3s6v9y$B&E)H+MbQeThWmZq4`)

func TestCryptionHandlerGet(t *testing.T) {
	handler := CryptionHandler(aesKey)(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.AppendBody([]byte(respText))
		ctx.Response.Header.Set("X-Test", "test")
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
	req.SetRequestURI("http://localhost/any")
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}

	expect, err := codec.EcbEncrypt(aesKey, []byte(respText))
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, "test", string(resp.Header.Peek("X-Test")))
	assert.Equal(t, base64.StdEncoding.EncodeToString(expect), string(resp.Body()))
}

func TestCryptionHandlerGet_badKey(t *testing.T) {
	handler := CryptionHandler(append(aesKey, aesKey...))(
		func(ctx *fasthttp.RequestCtx) {
			ctx.Response.AppendBody([]byte(respText))
			ctx.Response.Header.Set("X-Test", "test")
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
	req.SetRequestURI("http://localhost/any")
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode())
}

func TestCryptionHandlerPost(t *testing.T) {
	enc, err := codec.EcbEncrypt(aesKey, []byte(reqText))
	assert.Nil(t, err)

	handler := CryptionHandler(aesKey)(func(ctx *fasthttp.RequestCtx) {
		body := ctx.PostBody()
		assert.Equal(t, reqText, string(body))

		ctx.Response.AppendBody([]byte(respText))
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
	req.SetRequestURI("http://localhost/any")
	req.SetBody([]byte(base64.StdEncoding.EncodeToString(enc)))
	err = c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}

	expect, err := codec.EcbEncrypt(aesKey, []byte(respText))
	assert.Nil(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode())
	assert.Equal(t, base64.StdEncoding.EncodeToString(expect), string(resp.Body()))
}

func TestCryptionHandlerPostBadEncryption(t *testing.T) {
	enc, err := codec.EcbEncrypt(aesKey, []byte(reqText))
	assert.Nil(t, err)

	handler := CryptionHandler(aesKey)(nil)
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
	req.SetRequestURI("http://localhost/any")
	req.SetBody(enc)
	err = c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
}

func TestCryptionHandlerWriteHeader(t *testing.T) {
	handler := CryptionHandler(aesKey)(func(ctx *fasthttp.RequestCtx) {
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
	req.SetRequestURI("http://localhost/any")
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode())
}

func TestCryptionHandlerFlush(t *testing.T) {
	handler := CryptionHandler(aesKey)(func(ctx *fasthttp.RequestCtx) {
		ctx.Response.AppendBody([]byte(respText))
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
	req.SetRequestURI("http://localhost/any")
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}

	expect, err := codec.EcbEncrypt(aesKey, []byte(respText))
	assert.Nil(t, err)
	assert.Equal(t, base64.StdEncoding.EncodeToString(expect), string(resp.Body()))
}

//func TestCryptionHandler_Hijack(t *testing.T) {
//	resp := httptest.NewRecorder()
//	writer := newCryptionResponseWriter(resp)
//	assert.NotPanics(t, func() {
//		writer.Hijack()
//	})
//
//	writer = newCryptionResponseWriter(mockedHijackable{resp})
//	assert.NotPanics(t, func() {
//		writer.Hijack()
//	})
//}

func TestCryptionHandler_ContentTooLong(t *testing.T) {
	handler := CryptionHandler(aesKey)(func(ctx *fasthttp.RequestCtx) {
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
	req.SetRequestURI("http://localhost/")
	body := make([]byte, maxBytes+1)
	_, err := rand.Read(body)
	assert.NoError(t, err)
	req.SetBody(body)
	err = c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}

	assert.Nil(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode())
}

func TestCryptionHandler_BadBody(t *testing.T) {
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("http://localhost/foo")
	req.SetBodyStream(iotest.ErrReader(io.ErrUnexpectedEOF), -1)
	err := decryptBody(maxBytes, aesKey, req)
	assert.ErrorIs(t, err, io.ErrUnexpectedEOF)
}

// TODO
func TestCryptionHandler_BadKey(t *testing.T) {
	enc, err := codec.EcbEncrypt(aesKey, []byte(reqText))
	assert.Nil(t, err)

	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(fasthttp.MethodPost)
	req.SetRequestURI("http://localhost/any")
	req.SetBody([]byte(base64.StdEncoding.EncodeToString(enc)))
	err = decryptBody(maxBytes, append(aesKey, aesKey...), req)
	assert.Error(t, err)
}

//func TestCryptionResponseWriter_Flush(t *testing.T) {
//	body := []byte("hello, world!")
//
//	t.Run("half", func(t *testing.T) {
//		recorder := httptest.NewRecorder()
//		f := flushableResponseWriter{
//			writer: &halfWriter{recorder},
//		}
//		w := newCryptionResponseWriter(f)
//		_, err := w.Write(body)
//		assert.NoError(t, err)
//		w.flush(aesKey)
//		b, err := io.ReadAll(recorder.Body)
//		assert.NoError(t, err)
//		expected, err := codec.EcbEncrypt(aesKey, body)
//		assert.NoError(t, err)
//		assert.True(t, strings.HasPrefix(base64.StdEncoding.EncodeToString(expected), string(b)))
//		assert.True(t, len(string(b)) < len(base64.StdEncoding.EncodeToString(expected)))
//	})
//
//	t.Run("full", func(t *testing.T) {
//		recorder := httptest.NewRecorder()
//		f := flushableResponseWriter{
//			writer: recorder,
//		}
//		w := newCryptionResponseWriter(f)
//		_, err := w.Write(body)
//		assert.NoError(t, err)
//		w.flush(aesKey)
//		b, err := io.ReadAll(recorder.Body)
//		assert.NoError(t, err)
//		expected, err := codec.EcbEncrypt(aesKey, body)
//		assert.NoError(t, err)
//		assert.Equal(t, base64.StdEncoding.EncodeToString(expected), string(b))
//	})
//
//	t.Run("bad writer", func(t *testing.T) {
//		buf := logtest.NewCollector(t)
//		f := flushableResponseWriter{
//			writer: new(badWriter),
//		}
//		w := newCryptionResponseWriter(f)
//		_, err := w.Write(body)
//		assert.NoError(t, err)
//		w.flush(aesKey)
//		assert.True(t, strings.Contains(buf.Content(), io.ErrClosedPipe.Error()))
//	})
//}

type flushableResponseWriter struct {
	writer io.Writer
}

func (m flushableResponseWriter) Header() http.Header {
	panic("implement me")
}

func (m flushableResponseWriter) Write(p []byte) (int, error) {
	return m.writer.Write(p)
}

func (m flushableResponseWriter) WriteHeader(_ int) {
	panic("implement me")
}

type halfWriter struct {
	w io.Writer
}

func (t *halfWriter) Write(p []byte) (n int, err error) {
	n = len(p) >> 1
	return t.w.Write(p[0:n])
}

type badWriter struct {
}

func (b *badWriter) Write(_ []byte) (n int, err error) {
	return 0, io.ErrClosedPipe
}
