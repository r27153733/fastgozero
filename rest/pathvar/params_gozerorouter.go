//go:build gozerorouter

package pathvar

import (
	"github.com/r27153733/fastgozero/fastext/fastctx"
	"github.com/valyala/fasthttp"
)

var pathVars = contextKey("pathVars")

// Vars parses path variables and returns a map.
func Vars(r *fasthttp.RequestCtx) map[string]string {
	vars, ok := r.Value(pathVars).(map[string]string)
	if ok {
		return vars
	}

	return nil
}

// SetVars writes params into given r.
func SetVars(r *fasthttp.RequestCtx, params map[string]string) (free func()) {
	return fastctx.SetUserValueCtx(r, pathVars, params)
}

type contextKey string

func (c contextKey) String() string {
	return "rest/pathvar/context key: " + string(c)
}
