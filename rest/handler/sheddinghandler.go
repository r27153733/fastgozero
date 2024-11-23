package handler

import (
	"net/http"
	"sync"

	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/core/load"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stat"
	"github.com/zeromicro/go-zero/fastext"
	"github.com/zeromicro/go-zero/rest/httpx"
)

const serviceType = "api"

var (
	sheddingStat *load.SheddingStat
	lock         sync.Mutex
)

// SheddingHandler returns a middleware that does load shedding.
func SheddingHandler(shedder load.Shedder, metrics *stat.Metrics) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	if shedder == nil {
		return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return next
		}
	}

	ensureSheddingStat()

	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			sheddingStat.IncrementTotal()
			promise, err := shedder.Allow()
			if err != nil {
				metrics.AddDrop()
				sheddingStat.IncrementDrop()
				logx.Errorf("[http] dropped, %s - %s - %s",
					fastext.B2s(ctx.RequestURI()), httpx.GetRemoteAddr(ctx), fastext.B2s(ctx.UserAgent()))
				ctx.SetStatusCode(http.StatusServiceUnavailable)
				return
			}

			defer func() {
				if ctx.Response.StatusCode() == fasthttp.StatusServiceUnavailable {
					promise.Fail()
				} else {
					sheddingStat.IncrementPass()
					promise.Pass()
				}
			}()
			next(ctx)
		}
	}
}

func ensureSheddingStat() {
	lock.Lock()
	if sheddingStat == nil {
		sheddingStat = load.NewSheddingStat(serviceType)
	}
	lock.Unlock()
}
