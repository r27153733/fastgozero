package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/r27153733/fastgozero/core/breaker"
	"github.com/r27153733/fastgozero/core/logx"
	"github.com/r27153733/fastgozero/core/stat"
	"github.com/r27153733/fastgozero/fastext"
	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/valyala/fasthttp"
)

const breakerSeparator = "://"

// BreakerHandler returns a break circuit middleware.
func BreakerHandler(method, path string, metrics *stat.Metrics) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	brk := breaker.NewBreaker(breaker.WithName(strings.Join([]string{method, path}, breakerSeparator)))
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			promise, err := brk.Allow()
			if err != nil {
				metrics.AddDrop()
				logx.Errorf("[http] dropped, %s - %s - %s",
					fastext.B2s(ctx.RequestURI()), httpx.GetRemoteAddr(ctx), fastext.B2s(ctx.UserAgent()))
				ctx.SetStatusCode(fasthttp.StatusServiceUnavailable)
				return
			}

			defer func() {
				code := ctx.Response.StatusCode()
				if code < fasthttp.StatusInternalServerError {
					promise.Accept()
				} else {
					promise.Reject(fmt.Sprintf("%d %s", code, http.StatusText(code)))
				}
			}()
			next(ctx)
		}
	}
}
