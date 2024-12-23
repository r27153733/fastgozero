package cors

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestAddAllowHeaders(t *testing.T) {
	tests := []struct {
		name     string
		initial  string
		headers  []string
		expected string
	}{
		{
			name:     "single header",
			initial:  "",
			headers:  []string{"Content-Type"},
			expected: "Content-Type",
		},
		{
			name:     "multiple headers",
			initial:  "",
			headers:  []string{"Content-Type", "Authorization", "X-Requested-With"},
			expected: "Content-Type, Authorization, X-Requested-With",
		},
		{
			name:     "add to existing headers",
			initial:  "Origin, Accept",
			headers:  []string{"Content-Type"},
			expected: "Origin, Accept, Content-Type",
		},
		{
			name:     "no headers",
			initial:  "",
			headers:  []string{},
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			header := &fasthttp.ResponseHeader{}
			headers := make(map[string]struct{})
			if tt.initial != "" {
				header.Set(allowHeaders, tt.initial)
				vals := strings.Split(tt.initial, ", ")
				for _, v := range vals {
					headers[v] = struct{}{}
				}
			}
			for _, h := range tt.headers {
				headers[h] = struct{}{}
			}
			AddAllowHeaders(header, tt.headers...)
			var actual []string
			vals := header.PeekAll(allowHeaders)
			for _, v := range vals {
				bunch := strings.Split(string(v), ", ")
				for _, b := range bunch {
					if len(b) > 0 {
						actual = append(actual, b)
					}
				}
			}

			var expect []string
			for k := range headers {
				expect = append(expect, k)
			}
			assert.ElementsMatch(t, expect, actual)
		})
	}
}

func TestCorsHandlerWithOrigins(t *testing.T) {
	tests := []struct {
		name      string
		origins   []string
		reqOrigin string
		expect    string
	}{
		{
			name:   "allow all origins",
			expect: allOrigins,
		},
		{
			name:      "allow one origin",
			origins:   []string{"http://local"},
			reqOrigin: "http://local",
			expect:    "http://local",
		},
		{
			name:      "allow many origins",
			origins:   []string{"http://local", "http://remote"},
			reqOrigin: "http://local",
			expect:    "http://local",
		},
		{
			name:      "allow sub origins",
			origins:   []string{"local", "remote"},
			reqOrigin: "sub.local",
			expect:    "sub.local",
		},
		{
			name:      "allow all origins",
			reqOrigin: "http://local",
			expect:    "*",
		},
		{
			name:      "allow many origins with all mark",
			origins:   []string{"http://local", "http://remote", "*"},
			reqOrigin: "http://another",
			expect:    "http://another",
		},
		{
			name:      "not allow origin",
			origins:   []string{"http://local", "http://remote"},
			reqOrigin: "http://another",
		},
		{
			name:      "not safe origin",
			origins:   []string{"safe.com"},
			reqOrigin: "not-safe.com",
		},
	}

	methods := []string{
		http.MethodOptions,
		http.MethodGet,
		http.MethodPost,
	}

	for _, test := range tests {
		for _, method := range methods {
			test := test
			t.Run(test.name+"-handler", func(t *testing.T) {
				ctx := new(fasthttp.RequestCtx)
				ctx.Request.Header.SetMethod(method)
				ctx.Request.SetRequestURI("http://localhost")
				r := &ctx.Request
				r.Header.Set(originHeader, test.reqOrigin)

				handler := NotAllowedHandler(nil, test.origins...)
				handler(ctx)
				if method == http.MethodOptions {
					assert.Equal(t, http.StatusNoContent, ctx.Response.StatusCode())
				} else {
					assert.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
				}
				assert.Equal(t, test.expect, string(ctx.Response.Header.Peek(allowOrigin)))
			})
			t.Run(test.name+"-handler-custom", func(t *testing.T) {
				ctx := new(fasthttp.RequestCtx)
				ctx.Request.Header.SetMethod(method)
				ctx.Request.SetRequestURI("http://localhost")
				r := &ctx.Request
				r.Header.Set(originHeader, test.reqOrigin)

				handler := NotAllowedHandler(func(w *fasthttp.Response) {
					w.Header.Set("foo", "bar")
				}, test.origins...)
				handler(ctx)
				if method == http.MethodOptions {
					assert.Equal(t, http.StatusNoContent, ctx.Response.StatusCode())
				} else {
					assert.Equal(t, http.StatusNotFound, ctx.Response.StatusCode())
				}
				assert.Equal(t, test.expect, string(ctx.Response.Header.Peek(allowOrigin)))
				assert.Equal(t, "bar", string(ctx.Response.Header.Peek("foo")))
			})
		}
	}

	for _, test := range tests {
		for _, method := range methods {
			test := test
			t.Run(test.name+"-middleware", func(t *testing.T) {
				ctx := new(fasthttp.RequestCtx)
				ctx.Request.Header.SetMethod(method)
				ctx.Request.SetRequestURI("http://localhost")
				r := &ctx.Request
				r.Header.Set(originHeader, test.reqOrigin)

				handler := Middleware(nil, test.origins...)(func(ctx *fasthttp.RequestCtx) {
					ctx.SetStatusCode(http.StatusOK)
				})
				handler(ctx)
				if method == http.MethodOptions {
					assert.Equal(t, http.StatusNoContent, ctx.Response.StatusCode())
				} else {
					assert.Equal(t, http.StatusOK, ctx.Response.StatusCode())
				}
				assert.Equal(t, test.expect, string(ctx.Response.Header.Peek(allowOrigin)))
			})
			t.Run(test.name+"-middleware-custom", func(t *testing.T) {
				ctx := new(fasthttp.RequestCtx)
				ctx.Request.Header.SetMethod(method)
				ctx.Request.SetRequestURI("http://localhost")
				r := &ctx.Request
				r.Header.Set(originHeader, test.reqOrigin)

				handler := Middleware(func(header *fasthttp.ResponseHeader) {
					header.Set("foo", "bar")
				}, test.origins...)(func(ctx *fasthttp.RequestCtx) {
					ctx.SetStatusCode(http.StatusOK)
				})
				handler(ctx)
				if method == http.MethodOptions {
					assert.Equal(t, http.StatusNoContent, ctx.Response.StatusCode())
				} else {
					assert.Equal(t, http.StatusOK, ctx.Response.StatusCode())
				}
				assert.Equal(t, test.expect, string(ctx.Response.Header.Peek(allowOrigin)))
				assert.Equal(t, "bar", string(ctx.Response.Header.Peek("foo")))
			})
		}
	}
}
