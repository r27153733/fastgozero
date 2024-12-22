package pathvar

import (
	"github.com/r27153733/fastgozero/fastext/fastctx"
	"github.com/valyala/fasthttp"
)

// Vars parses path variables and returns a map.
func Vars(r *fasthttp.RequestCtx) Params {
	vars, ok := r.Value(pathVars).(Params)
	if ok {
		return vars
	}

	return EmptyParams{}
}

// SetVars writes params into given r.
func SetVars(r *fasthttp.RequestCtx, params Params) (free func()) {
	return fastctx.SetUserValueCtx(r, pathVars, params)
}
