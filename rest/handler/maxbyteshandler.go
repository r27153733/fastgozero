package handler

import (
	"github.com/r27153733/fastgozero/rest/internal"
	"github.com/valyala/fasthttp"
)

// MaxBytesHandler returns a middleware that limit reading of http request body.
func MaxBytesHandler(n int64) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	if n <= 0 {
		return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return next
		}
	}

	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			if int64(ctx.Request.Header.ContentLength()) > n {
				internal.Errorf(ctx, "request entity too large, limit is %d, but got %d, rejected with code %d",
					n, ctx.Request.Header.ContentLength(), fasthttp.StatusRequestEntityTooLarge)
				ctx.SetStatusCode(fasthttp.StatusRequestEntityTooLarge)
			} else {
				next(ctx)
			}
		}
	}
}
