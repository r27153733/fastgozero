package handler

import (
	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/core/syncx"
	"github.com/zeromicro/go-zero/rest/internal"
)

// MaxConnsHandler returns a middleware that limit the concurrent connections.
func MaxConnsHandler(n int) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	if n <= 0 {
		return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return next
		}
	}

	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		latch := syncx.NewAtomicLimit(n)

		return func(ctx *fasthttp.RequestCtx) {
			if latch.TryBorrow() {
				defer latch.Return()
				next(ctx)
			} else {
				internal.Errorf(ctx, "concurrent connections over %d, rejected with code %d",
					n, fasthttp.StatusServiceUnavailable)
				ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
			}
		}
	}
}
