package fastctx

import "github.com/valyala/fasthttp"

func SetUserValueCtx(ctx *fasthttp.RequestCtx, key, val any) (free func()) {
	value := ctx.UserValue(key)
	if value == nil {
		free = func() {
			ctx.RemoveUserValue(key)
		}
	} else {
		free = func() {
			ctx.SetUserValue(key, value)
		}
	}
	ctx.SetUserValue(key, val)
	return free
}
