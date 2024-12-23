package fileserver

import (
	"github.com/valyala/fasthttp"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		dir             string
		requestPath     string
		expectedStatus  int
		expectedContent string
	}{
		{
			name:            "Serve static file",
			path:            "/static/",
			dir:             "./testdata",
			requestPath:     "/static/example.txt",
			expectedStatus:  http.StatusOK,
			expectedContent: "1",
		},
		{
			name:           "Pass through non-matching path",
			path:           "/static/",
			dir:            "./testdata",
			requestPath:    "/other/path",
			expectedStatus: http.StatusAlreadyReported,
		},
		{
			name:            "Directory with trailing slash",
			path:            "/assets",
			dir:             "testdata",
			requestPath:     "/assets/sample.txt",
			expectedStatus:  http.StatusOK,
			expectedContent: "2",
		},
		{
			name:           "Not exist file",
			path:           "/assets",
			dir:            "testdata",
			requestPath:    "/assets/not-exist.txt",
			expectedStatus: http.StatusAlreadyReported,
		},
		{
			name:           "Not exist file in root",
			path:           "/",
			dir:            "testdata",
			requestPath:    "/not-exist.txt",
			expectedStatus: http.StatusAlreadyReported,
		},
		{
			name:           "websocket request",
			path:           "/",
			dir:            "testdata",
			requestPath:    "/ws",
			expectedStatus: http.StatusAlreadyReported,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			middleware := Middleware(tt.path, os.DirFS(tt.dir))
			nextHandler := func(ctx *fasthttp.RequestCtx) {
				ctx.SetStatusCode(http.StatusAlreadyReported)
			}

			handlerToTest := middleware(nextHandler)

			for i := 0; i < 2; i++ {
				ctx := new(fasthttp.RequestCtx)
				ctx.Request.Header.SetMethod(fasthttp.MethodGet)
				ctx.Request.SetRequestURI(tt.requestPath)

				handlerToTest(ctx)

				assert.Equal(t, tt.expectedStatus, ctx.Response.StatusCode())
				if len(tt.expectedContent) > 0 {
					assert.Equal(t, tt.expectedContent, string(ctx.Response.Body()))
				}
			}
		})
	}
}

func TestEnsureTrailingSlash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"path", "path/"},
		{"path/", "path/"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ensureTrailingSlash(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEnsureNoTrailingSlash(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"path", "path"},
		{"path/", "path"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ensureNoTrailingSlash(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
