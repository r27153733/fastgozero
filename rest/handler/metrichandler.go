package handler

import (
	"github.com/r27153733/fastgozero/core/stat"
	"github.com/r27153733/fastgozero/core/timex"
	"github.com/valyala/fasthttp"
)

// MetricHandler returns a middleware that stat the metrics.
func MetricHandler(metrics *stat.Metrics) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			startTime := timex.Now()
			defer func() {
				metrics.Add(stat.Task{
					Duration: timex.Since(startTime),
				})
			}()

			next(ctx)
		}
	}
}
