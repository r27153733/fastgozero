package handler

import (
	"fmt"
	"runtime/debug"

	"github.com/r27153733/fastgozero/rest/internal"
	"github.com/valyala/fasthttp"
)

// RecoverHandler returns a middleware that recovers if panic happens.
func RecoverHandler(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		defer func() {
			if result := recover(); result != nil {
				internal.Error(ctx, fmt.Sprintf("%v\n%s", result, debug.Stack()))
				ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			}
		}()

		next(ctx)
	}
}
