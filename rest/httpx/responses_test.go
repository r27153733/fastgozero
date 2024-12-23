package httpx

import (
	"context"
	"errors"
	"fmt"
	"github.com/valyala/fasthttp"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"net/http"
	"strings"
	"testing"

	"github.com/r27153733/fastgozero/core/logx"
	"github.com/stretchr/testify/assert"
)

type message struct {
	Name string `json:"name"`
}

func init() {
	logx.Disable()
}

func TestError(t *testing.T) {
	const (
		body        = "foo"
		wrappedBody = `"foo"`
	)

	tests := []struct {
		name          string
		input         string
		errorHandler  func(error) (int, any)
		expectHasBody bool
		expectBody    string
		expectCode    int
	}{
		{
			name:          "default error handler",
			input:         body,
			expectHasBody: true,
			expectBody:    body,
			expectCode:    http.StatusBadRequest,
		},
		{
			name:  "customized error handler return string",
			input: body,
			errorHandler: func(err error) (int, any) {
				return http.StatusForbidden, err.Error()
			},
			expectHasBody: true,
			expectBody:    wrappedBody,
			expectCode:    http.StatusForbidden,
		},
		{
			name:  "customized error handler return error",
			input: body,
			errorHandler: func(err error) (int, any) {
				return http.StatusForbidden, err
			},
			expectHasBody: true,
			expectBody:    body,
			expectCode:    http.StatusForbidden,
		},
		{
			name:  "customized error handler return nil",
			input: body,
			errorHandler: func(err error) (int, any) {
				return http.StatusForbidden, nil
			},
			expectHasBody: false,
			expectBody:    "",
			expectCode:    http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.errorHandler != nil {
				prev := errorHandler.Load()
				SetErrorHandler(test.errorHandler)
				defer func() {
					errorHandler.Store(prev)
				}()
			}
			resp := new(fasthttp.Response)
			Error(resp, errors.New(test.input))
			assert.Equal(t, test.expectCode, resp.StatusCode())
			assert.Equal(t, test.expectHasBody, len(resp.Body()) > 0)
			assert.Equal(t, test.expectBody, strings.TrimSpace(string(resp.Body())))
		})
	}
}

func TestErrorWithGrpcError(t *testing.T) {
	resp := new(fasthttp.Response)
	Error(resp, status.Error(codes.Unavailable, "foo"))
	assert.Equal(t, http.StatusServiceUnavailable, resp.StatusCode())
	assert.True(t, len(resp.Body()) > 0)
	assert.True(t, strings.Contains(string(resp.Body()), "foo"))
}

func TestErrorWithHandler(t *testing.T) {
	resp := new(fasthttp.Response)
	Error(resp, errors.New("foo"), func(resp *fasthttp.Response, err error) {
		resp.SetStatusCode(499)
		resp.Header.SetContentTypeBytes([]byte("text/plain; charset=utf-8"))
		resp.SetBodyString(err.Error())
	})
	assert.Equal(t, 499, resp.StatusCode())
	assert.True(t, len(resp.Body()) > 0)
	assert.Equal(t, "foo", strings.TrimSpace(string(resp.Body())))
}

//func TestOk(t *testing.T) {
//	w := tracedResponseWriter{
//		headers: make(map[string][]string),
//	}
//	Ok(&w)
//	assert.Equal(t, http.StatusOK, w.code)
//}

func TestOkJson(t *testing.T) {
	t.Run("no handler", func(t *testing.T) {
		resp := new(fasthttp.Response)
		msg := message{Name: "anyone"}
		OkJson(resp, msg)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Equal(t, "{\"name\":\"anyone\"}", string(resp.Body()))
	})

	t.Run("with handler", func(t *testing.T) {
		prev := okHandler.Load()
		t.Cleanup(func() {
			okHandler.Store(prev)
		})

		SetOkHandler(func(_ context.Context, v interface{}) any {
			return fmt.Sprintf("hello %s", v.(message).Name)
		})
		resp := new(fasthttp.Response)
		msg := message{Name: "anyone"}
		OkJson(resp, msg)
		assert.Equal(t, http.StatusOK, resp.StatusCode())
		assert.Equal(t, `"hello anyone"`, string(resp.Body()))
	})
}

func TestOkJsonCtx(t *testing.T) {
	t.Run("no handler", func(t *testing.T) {
		resp := new(fasthttp.RequestCtx)
		msg := message{Name: "anyone"}
		OkJsonCtx(resp, msg)
		assert.Equal(t, http.StatusOK, resp.Response.StatusCode())
		assert.Equal(t, "{\"name\":\"anyone\"}", string(resp.Response.Body()))
	})

	t.Run("with handler", func(t *testing.T) {
		prev := okHandler.Load()
		t.Cleanup(func() {
			okHandler.Store(prev)
		})

		SetOkHandler(func(_ context.Context, v interface{}) any {
			return fmt.Sprintf("hello %s", v.(message).Name)
		})
		resp := new(fasthttp.RequestCtx)
		msg := message{Name: "anyone"}
		OkJsonCtx(resp, msg)
		assert.Equal(t, http.StatusOK, resp.Response.StatusCode())
		assert.Equal(t, `"hello anyone"`, string(resp.Response.Body()))
	})
}

//func TestWriteJsonTimeout(t *testing.T) {
//	// only log it and ignore
//	w := tracedResponseWriter{
//		headers: make(map[string][]string),
//		err:     http.ErrHandlerTimeout,
//	}
//	msg := message{Name: "anyone"}
//	WriteJson(&w, http.StatusOK, msg)
//	assert.Equal(t, http.StatusOK, w.code)
//}

//func TestWriteJsonError(t *testing.T) {
//	// only log it and ignore
//	w := tracedResponseWriter{
//		headers: make(map[string][]string),
//		err:     errors.New("foo"),
//	}
//	msg := message{Name: "anyone"}
//	WriteJson(&w, http.StatusOK, msg)
//	assert.Equal(t, http.StatusOK, w.code)
//}
//
//func TestWriteJsonLessWritten(t *testing.T) {
//	w := tracedResponseWriter{
//		headers:     make(map[string][]string),
//		lessWritten: true,
//	}
//	msg := message{Name: "anyone"}
//	WriteJson(&w, http.StatusOK, msg)
//	assert.Equal(t, http.StatusOK, w.code)
//}
//
//func TestWriteJsonMarshalFailed(t *testing.T) {
//	w := tracedResponseWriter{
//		headers: make(map[string][]string),
//	}
//	WriteJson(&w, http.StatusOK, map[string]any{
//		"Data": complex(0, 0),
//	})
//	assert.Equal(t, http.StatusInternalServerError, w.code)
//}
//
//func TestStream(t *testing.T) {
//	t.Run("regular case", func(t *testing.T) {
//		channel := make(chan string)
//		go func() {
//			defer close(channel)
//			for index := 0; index < 5; index++ {
//				channel <- fmt.Sprintf("%d", index)
//			}
//		}()
//
//		w := httptest.NewRecorder()
//		Stream(context.Background(), w, func(w io.Writer) bool {
//			output, ok := <-channel
//			if !ok {
//				return false
//			}
//
//			outputBytes := bytes.NewBufferString(output)
//			_, err := w.Write(append(outputBytes.Bytes(), []byte("\n")...))
//			return err == nil
//		})
//
//		assert.Equal(t, http.StatusOK, w.Code)
//		assert.Equal(t, "0\n1\n2\n3\n4\n", w.Body.String())
//	})
//
//	t.Run("context done", func(t *testing.T) {
//		channel := make(chan string)
//		go func() {
//			defer close(channel)
//			for index := 0; index < 5; index++ {
//				channel <- fmt.Sprintf("num: %d", index)
//			}
//		}()
//
//		w := httptest.NewRecorder()
//		ctx, cancel := context.WithCancel(context.Background())
//		cancel()
//		Stream(ctx, w, func(w io.Writer) bool {
//			output, ok := <-channel
//			if !ok {
//				return false
//			}
//
//			outputBytes := bytes.NewBufferString(output)
//			_, err := w.Write(append(outputBytes.Bytes(), []byte("\n")...))
//			return err == nil
//		})
//
//		assert.Equal(t, http.StatusOK, w.Code)
//		assert.Equal(t, "", w.Body.String())
//	})
//}
//
//type tracedResponseWriter struct {
//	headers     map[string][]string
//	builder     strings.Builder
//	hasBody     bool
//	code        int
//	lessWritten bool
//	wroteHeader bool
//	err         error
//}
//
//func (w *tracedResponseWriter) Header() http.Header {
//	return w.headers
//}
//
//func (w *tracedResponseWriter) Write(bytes []byte) (n int, err error) {
//	if w.err != nil {
//		return 0, w.err
//	}
//
//	n, err = w.builder.Write(bytes)
//	if w.lessWritten {
//		n--
//	}
//	w.hasBody = true
//
//	return
//}
//
//func (w *tracedResponseWriter) WriteHeader(code int) {
//	if w.wroteHeader {
//		return
//	}
//	w.wroteHeader = true
//	w.code = code
//}

func TestErrorCtx(t *testing.T) {
	const (
		body        = "foo"
		wrappedBody = `"foo"`
	)

	tests := []struct {
		name            string
		input           string
		errorHandlerCtx func(context.Context, error) (int, any)
		expectHasBody   bool
		expectBody      string
		expectCode      int
	}{
		{
			name:          "default error handler",
			input:         body,
			expectHasBody: true,
			expectBody:    body,
			expectCode:    http.StatusBadRequest,
		},
		{
			name:  "customized error handler return string",
			input: body,
			errorHandlerCtx: func(ctx context.Context, err error) (int, any) {
				return http.StatusForbidden, err.Error()
			},
			expectHasBody: true,
			expectBody:    wrappedBody,
			expectCode:    http.StatusForbidden,
		},
		{
			name:  "customized error handler return error",
			input: body,
			errorHandlerCtx: func(ctx context.Context, err error) (int, any) {
				return http.StatusForbidden, err
			},
			expectHasBody: true,
			expectBody:    body,
			expectCode:    http.StatusForbidden,
		},
		{
			name:  "customized error handler return nil",
			input: body,
			errorHandlerCtx: func(context.Context, error) (int, any) {
				return http.StatusForbidden, nil
			},
			expectHasBody: false,
			expectBody:    "",
			expectCode:    http.StatusForbidden,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx := new(fasthttp.RequestCtx)
			if test.errorHandlerCtx != nil {
				prev := errorHandler.Load()
				SetErrorHandlerCtx(test.errorHandlerCtx)
				defer func() {
					errorHandler.Store(prev)
				}()
			}
			ErrorCtx(ctx, errors.New(test.input))
			assert.Equal(t, test.expectCode, ctx.Response.StatusCode())
			assert.Equal(t, test.expectBody, strings.TrimSpace(string(ctx.Response.Body())))
		})
	}

	// The current handler is a global event,Set default values to avoid impacting subsequent unit tests
	SetErrorHandlerCtx(nil)
}

func TestErrorWithGrpcErrorCtx(t *testing.T) {
	ctx := new(fasthttp.RequestCtx)
	ErrorCtx(ctx, status.Error(codes.Unavailable, "foo"))
	assert.Equal(t, http.StatusServiceUnavailable, ctx.Response.StatusCode())
	assert.True(t, strings.Contains(string(ctx.Response.Body()), "foo"))
}

func TestErrorWithHandlerCtx(t *testing.T) {
	ctx := new(fasthttp.RequestCtx)
	ErrorCtx(ctx, errors.New("foo"), func(w *fasthttp.Response, err error) {
		w.SetBodyString(err.Error())
		w.SetStatusCode(499)
	})
	assert.Equal(t, 499, ctx.Response.StatusCode())
	assert.Equal(t, "foo", strings.TrimSpace(string(ctx.Response.Body())))
}

func TestWriteJsonCtxMarshalFailed(t *testing.T) {
	ctx := new(fasthttp.RequestCtx)
	WriteJsonCtx(ctx, http.StatusOK, map[string]any{
		"Data": complex(0, 0),
	})
	assert.Equal(t, http.StatusInternalServerError, ctx.Response.StatusCode())
}
