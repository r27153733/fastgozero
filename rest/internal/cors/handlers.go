package cors

import (
	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/fastext"
	"net/http"
	"strings"
)

const (
	allowOrigin      = "Access-Control-Allow-Origin"
	allOrigins       = "*"
	allowMethods     = "Access-Control-Allow-Methods"
	allowHeaders     = "Access-Control-Allow-Headers"
	allowCredentials = "Access-Control-Allow-Credentials"
	exposeHeaders    = "Access-Control-Expose-Headers"
	requestMethod    = "Access-Control-Request-Method"
	requestHeaders   = "Access-Control-Request-Headers"
	allowHeadersVal  = "Content-Type, Origin, X-CSRF-Token, Authorization, AccessToken, Token, Range"
	exposeHeadersVal = "Content-Length, Access-Control-Allow-Origin, Access-Control-Allow-Headers"
	methods          = "GET, HEAD, POST, PATCH, PUT, DELETE"
	allowTrue        = "true"
	maxAgeHeader     = "Access-Control-Max-Age"
	maxAgeHeaderVal  = "86400"
	varyHeader       = "Vary"
	originHeader     = "Origin"
)

// AddAllowHeaders sets the allowed headers.
func AddAllowHeaders(header *fasthttp.ResponseHeader, headers ...string) {
	header.Add(allowHeaders, strings.Join(headers, ", "))
}

// NotAllowedHandler handles cross domain not allowed requests.
// At most one origin can be specified, other origins are ignored if given, default to be *.
func NotAllowedHandler(fn func(w *fasthttp.Response), origins ...string) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		checkAndSetHeaders(ctx, origins)
		if fn != nil {
			fn(&ctx.Response)
		}

		if ctx.IsOptions() {
			ctx.SetStatusCode(http.StatusNoContent)
		} else {
			ctx.SetStatusCode(http.StatusNotFound)
		}
	}
}

// Middleware returns a middleware that adds CORS headers to the response.
func Middleware(fn func(w *fasthttp.ResponseHeader), origins ...string) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			checkAndSetHeaders(ctx, origins)
			if fn != nil {
				fn(&ctx.Response.Header)
			}

			if ctx.IsOptions() {
				ctx.SetStatusCode(http.StatusNoContent)
			} else {
				next(ctx)
			}
		}
	}
}

func checkAndSetHeaders(ctx *fasthttp.RequestCtx, origins []string) {
	setVaryHeaders(ctx)

	if len(origins) == 0 {
		setHeader(&ctx.Response, allOrigins)
		return
	}

	origin := fastext.B2s(ctx.Request.Header.Peek(originHeader))
	if isOriginAllowed(origins, origin) {
		setHeader(&ctx.Response, origin)
	}
}

func isOriginAllowed(allows []string, origin string) bool {
	origin = strings.ToLower(origin)

	for _, allow := range allows {
		if allow == allOrigins {
			return true
		}

		allow = strings.ToLower(allow)
		if origin == allow {
			return true
		}

		if strings.HasSuffix(origin, "."+allow) {
			return true
		}
	}

	return false
}

func setHeader(w *fasthttp.Response, origin string) {
	w.Header.Set(allowOrigin, origin)
	w.Header.Set(allowMethods, methods)
	w.Header.Set(allowHeaders, allowHeadersVal)
	w.Header.Set(exposeHeaders, exposeHeadersVal)
	if origin != allOrigins {
		w.Header.Set(allowCredentials, allowTrue)
	}
	w.Header.Set(maxAgeHeader, maxAgeHeaderVal)
}

func setVaryHeaders(ctx *fasthttp.RequestCtx) {
	ctx.Response.Header.Add(varyHeader, originHeader)
	if ctx.IsOptions() {
		ctx.Response.Header.Add(varyHeader, requestMethod)
		ctx.Response.Header.Add(varyHeader, requestHeaders)
	}
}
