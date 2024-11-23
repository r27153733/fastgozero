package httpx

import (
	"github.com/valyala/fasthttp"
)

// Router interface represents a http router that handles http requests.
type Router interface {
	ServeHTTP(ctx *fasthttp.RequestCtx)
	Handle(method, path string, handler fasthttp.RequestHandler) error
	SetNotFoundHandler(handler fasthttp.RequestHandler)
	SetNotAllowedHandler(handler fasthttp.RequestHandler)
}
