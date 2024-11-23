package rest

import (
	"github.com/valyala/fasthttp"
	"time"
)

type (
	// Middleware defines the middleware method.
	Middleware func(next fasthttp.RequestHandler) fasthttp.RequestHandler

	// A Route is a http route.
	Route struct {
		Method  string
		Path    string
		Handler fasthttp.RequestHandler
	}

	// RouteOption defines the method to customize a featured route.
	RouteOption func(r *featuredRoutes)

	jwtSetting struct {
		enabled    bool
		secret     string
		prevSecret string
	}

	signatureSetting struct {
		SignatureConf
		enabled bool
	}

	featuredRoutes struct {
		timeout   time.Duration
		priority  bool
		jwt       jwtSetting
		signature signatureSetting
		routes    []Route
		maxBytes  int64
	}
)
