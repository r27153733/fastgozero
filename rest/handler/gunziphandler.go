package handler

import (
	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/fastext"
)

const gzipEncoding = "gzip"

// GunzipHandler returns a middleware to gunzip http request body.
func GunzipHandler(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if fastext.B2s(ctx.Request.Header.ContentEncoding()) == "gzip" {
			//reader, err := ctx.Request.BodyGunzip()
			reader, err := fasthttp.AppendGunzipBytes(ctx.Response.Body()[0:], ctx.Response.Body())
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return
			}
			ctx.SetBody(reader)
		}

		next(ctx)
	}
}
