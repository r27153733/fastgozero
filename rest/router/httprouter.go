//go:build !gozerorouter

package router

import (
	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/r27153733/fastgozero/rest/router/internal/httprouter"
)

var (
	// ErrInvalidMethod is an error that indicates not a valid http method.
	ErrInvalidMethod = httprouter.ErrInvalidMethod
	// ErrInvalidPath is an error that indicates path is not start with /.
	ErrInvalidPath = httprouter.ErrInvalidPath
)

// NewRouter returns a httpx.Router.
func NewRouter() httpx.Router {
	r := httprouter.New()
	r.RemoveExtraSlash = true
	r.RedirectTrailingSlash = false
	return r
}
