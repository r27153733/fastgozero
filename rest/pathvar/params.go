//go:build !gozerorouter

package pathvar

import (
	"github.com/r27153733/fastgozero/fastext/fastctx"
	"github.com/r27153733/fastgozero/rest/router/httprouter"
	"github.com/valyala/fasthttp"
)

var pathVars = contextKey("pathVars")

// Vars parses path variables and returns a map.
func Vars(r *fasthttp.RequestCtx) map[string]string {
	value := httprouter.ParamsFromContext(r)
	if value != nil {
		m := map[string]string{}
		for _, n := range *value {
			m[n.Key] = n.Value
		}
		return m
	}
	return nil
}

// SetVars writes params into given r.
func SetVars(r *fasthttp.RequestCtx, params map[string]string) (free func()) {
	var httprouterParams httprouter.Params
	for k, v := range params {
		httprouterParams = append(httprouterParams, httprouter.Param{Key: k, Value: v})
	}
	return fastctx.SetUserValueCtx(r, httprouter.ParamsKey, httprouterParams)
}

type contextKey string

func (c contextKey) String() string {
	return "rest/pathvar/context key: " + string(c)
}
