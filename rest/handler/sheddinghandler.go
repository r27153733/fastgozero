package handler

import (
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"net/http"
	"sync"

	"github.com/r27153733/fastgozero/core/load"
	"github.com/r27153733/fastgozero/core/logx"
	"github.com/r27153733/fastgozero/core/stat"
	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/valyala/fasthttp"
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
					bytesconv.BToS(ctx.RequestURI()), httpx.GetRemoteAddr(ctx), bytesconv.BToS(ctx.UserAgent()))
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
