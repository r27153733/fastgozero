package handler

import (
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"github.com/valyala/fasthttp"
)

const gzipEncoding = "gzip"

// GunzipHandler returns a middleware to gunzip http request body.
func GunzipHandler(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if bytesconv.BToS(ctx.Request.Header.ContentEncoding()) == "gzip" {
			reader, err := fasthttp.AppendGunzipBytes(ctx.Request.Body()[:0], ctx.Request.Body())
			if err != nil {
				ctx.SetStatusCode(fasthttp.StatusBadRequest)
				return
			}
			ctx.Request.SetBody(reader)
		}

		next(ctx)
	}
}
