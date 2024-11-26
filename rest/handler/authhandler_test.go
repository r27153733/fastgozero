package handler

import (
	"bufio"
	"net"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

func TestAuthHandlerFailed(t *testing.T) {
	handler := Authorize("B63F477D-BBA3-4E52-96D3-C0034C27694A", WithUnauthorizedCallback(
		func(ctx *fasthttp.RequestCtx, err error) {
			assert.NotNil(t, err)
			ctx.Response.Header.Set("X-Test", err.Error())
			ctx.Response.SetStatusCode(fasthttp.StatusUnauthorized)
			ctx.Response.AppendBody([]byte("content"))
		}))(
		func(ctx *fasthttp.RequestCtx) {
			ctx.Response.SetStatusCode(fasthttp.StatusOK)
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
	req.SetRequestURI("http://example.com")
	resp := fasthttp.AcquireResponse()
	err := c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, fasthttp.StatusUnauthorized, resp.StatusCode())
}

func TestAuthHandler(t *testing.T) {
	const key = "B63F477D-BBA3-4E52-96D3-C0034C27694A"
	handler := Authorize(key)(
		func(ctx *fasthttp.RequestCtx) {
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
	token, err := buildToken(key, map[string]any{
		"key": "value",
	}, 3600)
	assert.Nil(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	err = c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	assert.Equal(t, "test", string(resp.Header.Peek("X-Test")))
	assert.Equal(t, "content", string(resp.Body()))
}

func TestAuthHandlerWithPrevSecret(t *testing.T) {
	const (
		key     = "14F17379-EB8F-411B-8F12-6929002DCA76"
		prevKey = "B63F477D-BBA3-4E52-96D3-C0034C27694A"
	)
	handler := Authorize(key, WithPrevSecret(prevKey))(
		func(ctx *fasthttp.RequestCtx) {
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
	token, err := buildToken(key, map[string]any{
		"key": "value",
	}, 3600)
	assert.Nil(t, err)
	req.Header.Set("Authorization", "Bearer "+token)
	err = c.Do(req, resp)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	assert.Equal(t, "test", string(resp.Header.Peek("X-Test")))
	assert.Equal(t, "content", string(resp.Body()))
}

func TestAuthHandler_NilError(t *testing.T) {
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://localhost")
	assert.NotPanics(t, func() {
		unauthorized(ctx, nil, nil)
	})
}

func buildToken(secretKey string, payloads map[string]any, seconds int64) (string, error) {
	now := time.Now().Unix()
	claims := make(jwt.MapClaims)
	claims["exp"] = now + seconds
	claims["iat"] = now
	for k, v := range payloads {
		claims[k] = v
	}

	token := jwt.New(jwt.SigningMethodHS256)
	token.Claims = claims

	return token.SignedString([]byte(secretKey))
}

type mockedHijackable struct {
	*httptest.ResponseRecorder
}

func (m mockedHijackable) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	return nil, nil, nil
}
