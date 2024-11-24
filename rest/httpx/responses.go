package httpx

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"github.com/valyala/fasthttp"
	"io"
	"net/http"
	"sync"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/internal/errcode"
	"github.com/zeromicro/go-zero/rest/internal/header"
)

var (
	errorHandler func(context.Context, error) (int, any)
	errorLock    sync.RWMutex
	okHandler    func(context.Context, any) any
	okLock       sync.RWMutex
)

// Error writes err into w.
func Error(w *fasthttp.Response, err error, fns ...func(w *fasthttp.Response, err error)) {
	doHandleError(w, err, buildErrorHandler(context.Background()), WriteJson, fns...)
}

// ErrorCtx writes err into w.
func ErrorCtx(ctx *fasthttp.RequestCtx, err error,
	fns ...func(w *fasthttp.Response, err error)) {
	writeJson := func(w *fasthttp.Response, code int, v any) {
		WriteJsonCtx(ctx, code, v)
	}
	doHandleError(&ctx.Response, err, buildErrorHandler(ctx), writeJson, fns...)
}

// Ok writes HTTP 200 OK into w.
func Ok(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
}

// OkJson writes v into w with 200 OK.
func OkJson(w *fasthttp.Response, v any) {
	okLock.RLock()
	handler := okHandler
	okLock.RUnlock()
	if handler != nil {
		v = handler(context.Background(), v)
	}
	WriteJson(w, http.StatusOK, v)
}

// OkJsonCtx writes v into w with 200 OK.
func OkJsonCtx(ctx *fasthttp.RequestCtx, v any) {
	okLock.RLock()
	handlerCtx := okHandler
	okLock.RUnlock()
	if handlerCtx != nil {
		v = handlerCtx(ctx, v)
	}
	WriteJsonCtx(ctx, http.StatusOK, v)
}

// SetErrorHandler sets the error handler, which is called on calling Error.
// Notice: SetErrorHandler and SetErrorHandlerCtx set the same error handler.
// Keeping both SetErrorHandler and SetErrorHandlerCtx is for backward compatibility.
func SetErrorHandler(handler func(error) (int, any)) {
	errorLock.Lock()
	defer errorLock.Unlock()
	errorHandler = func(_ context.Context, err error) (int, any) {
		return handler(err)
	}
}

// SetErrorHandlerCtx sets the error handler, which is called on calling Error.
// Notice: SetErrorHandler and SetErrorHandlerCtx set the same error handler.
// Keeping both SetErrorHandler and SetErrorHandlerCtx is for backward compatibility.
func SetErrorHandlerCtx(handlerCtx func(context.Context, error) (int, any)) {
	errorLock.Lock()
	defer errorLock.Unlock()
	errorHandler = handlerCtx
}

// SetOkHandler sets the response handler, which is called on calling OkJson and OkJsonCtx.
func SetOkHandler(handler func(context.Context, any) any) {
	okLock.Lock()
	defer okLock.Unlock()
	okHandler = handler
}

// Stream writes data into w with streaming mode.
// The ctx is used to control the streaming loop, typically use r.Context().
// The fn is called repeatedly until it returns false.
func Stream(ctx *fasthttp.RequestCtx, fn func(w io.Writer) bool) {
	wg := sync.WaitGroup{}
	wg.Add(1)
	ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
		defer wg.Done()
		for {
			select {
			case <-ctx.Done():
				return
			default:
				hasMore := fn(w)
				if !hasMore {
					return
				}
			}
		}
	})
	wg.Wait()
}

// WriteJson writes v as json string into w with code.
func WriteJson(w *fasthttp.Response, code int, v any) {
	if err := doWriteJson(w, code, v); err != nil {
		logx.Error(err)
	}
}

// WriteJsonCtx writes v as json string into w with code.
func WriteJsonCtx(ctx *fasthttp.RequestCtx, code int, v any) {
	if err := doWriteJson(&ctx.Response, code, v); err != nil {
		logx.WithContext(ctx).Error(err)
	}
}

func buildErrorHandler(ctx context.Context) func(error) (int, any) {
	errorLock.RLock()
	handlerCtx := errorHandler
	errorLock.RUnlock()

	var handler func(error) (int, any)
	if handlerCtx != nil {
		handler = func(err error) (int, any) {
			return handlerCtx(ctx, err)
		}
	}

	return handler
}

func doHandleError(w *fasthttp.Response, err error, handler func(error) (int, any),
	writeJson func(w *fasthttp.Response, code int, v any),
	fns ...func(w *fasthttp.Response, err error)) {
	if handler == nil {
		if len(fns) > 0 {
			for _, fn := range fns {
				fn(w, err)
			}
		} else if errcode.IsGrpcError(err) {
			// don't unwrap error and get status.Message(),
			// it hides the rpc error headers.
			w.Reset()
			w.SetStatusCode(errcode.CodeFromGrpcError(err))
			w.Header.SetContentTypeBytes([]byte("text/plain; charset=utf-8"))
			w.SetBodyString(err.Error())
		} else {
			w.Reset()
			w.SetStatusCode(fasthttp.StatusBadRequest)
			w.Header.SetContentTypeBytes([]byte("text/plain; charset=utf-8"))
			w.SetBodyString(err.Error())
		}

		return
	}

	code, body := handler(err)
	if body == nil {
		w.SetStatusCode(code)
		return
	}

	switch v := body.(type) {
	case error:
		w.Reset()
		w.SetStatusCode(code)
		w.Header.SetContentTypeBytes([]byte("text/plain; charset=utf-8"))
		w.SetBodyString(v.Error())
	default:
		writeJson(w, code, body)
	}
}

func doWriteJson(w *fasthttp.Response, code int, v any) error {
	bs, err := json.Marshal(v)
	if err != nil {
		w.Reset()
		w.SetStatusCode(fasthttp.StatusInternalServerError)
		w.Header.SetContentTypeBytes([]byte("text/plain; charset=utf-8"))
		w.SetBodyString(err.Error())
		return fmt.Errorf("marshal json failed, error: %w", err)
	}

	w.Header.Set(ContentType, header.JsonContentType)
	w.SetStatusCode(code)
	w.AppendBody(bs)
	w.Body()
	return nil
}
