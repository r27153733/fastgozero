package handler

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
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

	req := httptest.NewRequest(fasthttp.MethodGet, "http://localhost", http.NoBody)
	req.Header.Set("X-Test", "test")
	resp := httptest.NewRecorder()
	handler()
	assert.Equal(t, fasthttp.StatusOK, resp.Code)
	assert.Equal(t, "test", resp.Header().Get("X-Test"))
	assert.Equal(t, "content", resp.Body.String())
}

func TestBreakerHandlerFail(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	breakerHandler := BreakerHandler(http.MethodGet, "/", metrics)
	handler := breakerHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://localhost", http.NoBody)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	assert.Equal(t, http.StatusBadGateway, resp.Code)
}

func TestBreakerHandler_4XX(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	breakerHandler := BreakerHandler(http.MethodGet, "/", metrics)
	handler := breakerHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	for i := 0; i < 1000; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost", http.NoBody)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
	}

	const tries = 100
	var pass int
	for i := 0; i < tries; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost", http.NoBody)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		if resp.Code == http.StatusBadRequest {
			pass++
		}
	}

	assert.Equal(t, tries, pass)
}

func TestBreakerHandlerReject(t *testing.T) {
	metrics := stat.NewMetrics("unit-test")
	breakerHandler := BreakerHandler(http.MethodGet, "/", metrics)
	handler := breakerHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))

	for i := 0; i < 1000; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost", http.NoBody)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
	}

	var drops int
	for i := 0; i < 100; i++ {
		req := httptest.NewRequest(http.MethodGet, "http://localhost", http.NoBody)
		resp := httptest.NewRecorder()
		handler.ServeHTTP(resp, req)
		if resp.Code == http.StatusServiceUnavailable {
			drops++
		}
	}

	assert.True(t, drops >= 80, fmt.Sprintf("expected to be greater than 80, but got %d", drops))
}
