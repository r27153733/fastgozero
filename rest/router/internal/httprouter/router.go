// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

// Package httprouter is a trie based high performance HTTP request router.
package httprouter

import (
	"context"
	"errors"
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"github.com/valyala/fasthttp"
	"net/http"
	"strings"
	"sync"
)

// Param is a single URL parameter, consisting of a key and a value.
type Param struct {
	Key   string
	Value string
}

// Params is a Param-slice, as returned by the router.
// The slice is ordered, the first URL parameter is also the first slice value.
// It is therefore safe to read values by the index.
type Params []Param

func (ps Params) VisitAll(f func(key, value string)) {
	for i := 0; i < len(ps); i++ {
		f(ps[i].Key, ps[i].Value)
	}
}

func (ps Params) Len() int {
	return len(ps)
}

// Get returns the value of the first Param which key matches the given name and a boolean true.
// If no matching Param is found, an empty string is returned and a boolean false .
func (ps Params) Get(name string) (string, bool) {
	for _, entry := range ps {
		if entry.Key == name {
			return entry.Value, true
		}
	}
	return "", false
}

// ByName returns the value of the first Param which key matches the given name.
// If no matching Param is found, an empty string is returned.
func (ps Params) ByName(name string) string {
	for _, p := range ps {
		if p.Key == name {
			return p.Value
		}
	}
	return ""
}

type paramsKey struct{}

// ParamsKey is the request context key under which URL params are stored.
var ParamsKey = paramsKey{}

// ParamsFromContext pulls the URL parameters from a request context,
// or returns nil if none are present.
func ParamsFromContext(ctx context.Context) *Params {
	p, _ := ctx.Value(ParamsKey).(*Params)
	return p
}

// Router is a http.Handler which can be used to dispatch requests to different
// handler functions via configurable routes
type Router struct {
	trees methodTrees

	paramsPool      sync.Pool
	skippedNodePool sync.Pool
	maxParams       uint16
	maxSections     uint16

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 308 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	// RemoveExtraSlash a parameter can be parsed from the URL even with extra slashes.
	// See the PR #1817 and issue #1644
	RemoveExtraSlash bool

	// Configurable http.Handler which is called when no matching route is
	// found. If it is not set, http.NotFound is used.
	NotFound fasthttp.RequestHandler

	// Configurable http.Handler which is called when a request
	// cannot be routed and HandleMethodNotAllowed is true.
	// If it is not set, http.Error with http.StatusMethodNotAllowed is used.
	// The "Allow" header with allowed request methods is set before the handler
	// is called.
	MethodNotAllowed fasthttp.RequestHandler
}

func (r *Router) SetNotFoundHandler(handler fasthttp.RequestHandler) {
	r.NotFound = handler
}

func (r *Router) SetNotAllowedHandler(handler fasthttp.RequestHandler) {
	r.MethodNotAllowed = handler
}

var _ fasthttp.RequestHandler = New().ServeHTTP

// New returns a new initialized Router.
// Path autocorrection, including trailing slashes, is enabled by default.
func New() *Router {
	return &Router{
		RedirectTrailingSlash:  true,
		HandleMethodNotAllowed: true,
	}
}

func (r *Router) getParams() *Params {
	ps, _ := r.paramsPool.Get().(*Params)
	if ps == nil && r.maxParams > 0 {
		tmp := make(Params, 0, r.maxParams)
		ps = &tmp
	}

	return ps
}

func (r *Router) putParams(ps *Params) {
	if ps != nil {
		*ps = (*ps)[:0]
		r.paramsPool.Put(ps)
	}
}

func (r *Router) getSkippedNodes() *[]skippedNode {
	sn, _ := r.skippedNodePool.Get().(*[]skippedNode)
	if sn == nil && r.maxSections > 0 {
		tmp := make([]skippedNode, 0, r.maxSections)
		sn = &tmp
	}

	return sn
}

func (r *Router) putSkippedNodes(sn *[]skippedNode) {
	if sn != nil {
		*sn = (*sn)[:0]
		r.skippedNodePool.Put(sn)
	}
}

// GET is a shortcut for router.Handle(http.MethodGet, path, handle)
func (r *Router) GET(path string, handle fasthttp.RequestHandler) {
	err := r.Handle(http.MethodGet, path, handle)
	if err != nil {
		panic(err)
	}
}

// HEAD is a shortcut for router.Handle(http.MethodHead, path, handle)
func (r *Router) HEAD(path string, handle fasthttp.RequestHandler) {
	err := r.Handle(http.MethodHead, path, handle)
	if err != nil {
		panic(err)
	}
}

// OPTIONS is a shortcut for router.Handle(http.MethodOptions, path, handle)
func (r *Router) OPTIONS(path string, handle fasthttp.RequestHandler) {
	err := r.Handle(http.MethodOptions, path, handle)
	if err != nil {
		panic(err)
	}
}

// POST is a shortcut for router.Handle(http.MethodPost, path, handle)
func (r *Router) POST(path string, handle fasthttp.RequestHandler) {
	err := r.Handle(http.MethodPost, path, handle)
	if err != nil {
		panic(err)
	}
}

// PUT is a shortcut for router.Handle(http.MethodPut, path, handle)
func (r *Router) PUT(path string, handle fasthttp.RequestHandler) {
	err := r.Handle(http.MethodPut, path, handle)
	if err != nil {
		panic(err)
	}
}

// PATCH is a shortcut for router.Handle(http.MethodPatch, path, handle)
func (r *Router) PATCH(path string, handle fasthttp.RequestHandler) {
	err := r.Handle(http.MethodPatch, path, handle)
	if err != nil {
		panic(err)
	}
}

// DELETE is a shortcut for router.Handle(http.MethodDelete, path, handle)
func (r *Router) DELETE(path string, handle fasthttp.RequestHandler) {
	err := r.Handle(http.MethodDelete, path, handle)
	if err != nil {
		panic(err)
	}
}

func validMethod(method string) bool {
	return method == fasthttp.MethodDelete || method == fasthttp.MethodGet ||
		method == fasthttp.MethodHead || method == fasthttp.MethodOptions ||
		method == fasthttp.MethodPatch || method == fasthttp.MethodPost ||
		method == fasthttp.MethodPut || method == fasthttp.MethodConnect ||
		method == fasthttp.MethodTrace
}

// Handle registers a new request handle with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (r *Router) Handle(method, path string, handler fasthttp.RequestHandler) error {
	if !validMethod(method) {
		return ErrInvalidMethod
	}
	if len(path) < 1 || path[0] != '/' {
		return ErrInvalidPath
	}
	if handler == nil {
		return errors.New("handle must not be nil")
	}

	if r.RemoveExtraSlash && len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	if r.trees == nil {
		r.trees = make(methodTrees, 0, 9)
	}

	root := r.trees.get(method)
	if root == nil {
		root = new(node)
		root.fullPath = "/"
		r.trees = append(r.trees, methodTree{method: method, root: root})
		//r.globalAllowed = r.allowed("/*", "")
	}

	root.addRoute(path, handler)

	// Update maxParams
	if paramsCount := countParams(path); paramsCount > r.maxParams {
		r.maxParams = paramsCount
	}

	if sectionsCount := countSections(path); sectionsCount > r.maxSections {
		r.maxSections = sectionsCount
	}

	return nil
}

var (
	// ErrInvalidMethod is an error that indicates not a valid http method.
	ErrInvalidMethod = errors.New("not a valid http method")
	// ErrInvalidPath is an error that indicates path is not start with /.
	ErrInvalidPath = errors.New("path must begin with '/'")
)

// Lookup allows the manual lookup of a method + path combo.
// This is e.g. useful to build a framework around this router.
// If the path was found, it returns the handle function and the path parameter
// values. Otherwise the third return value indicates whether a redirection to
// the same path with an extra / without the trailing slash should be performed.
//func (r *Router) Lookup(method, path string) (fasthttp.RequestHandler, Params, bool) {
//	if root := r.trees[method]; root != nil {
//		handle, ps, tsr := root.getValue(path, r.getParams)
//		if handle == nil {
//			r.putParams(ps)
//			return nil, nil, tsr
//		}
//		if ps == nil {
//			return handle, nil, tsr
//		}
//		return handle, *ps, tsr
//	}
//	return nil, nil, false
//}

// ServeHTTP makes the router implement the http.Handler interface.
func (r *Router) ServeHTTP(ctx *fasthttp.RequestCtx) {
	path := bytesconv.BToS(ctx.URI().Path())
	method := bytesconv.BToS(ctx.Method())

	if r.RemoveExtraSlash && len(path) > 1 && path[len(path)-1] == '/' {
		path = path[:len(path)-1]
	}

	if root := r.trees.get(method); root != nil {
		ps := r.getParams()
		defer r.putParams(ps)
		skippedNodes := r.getSkippedNodes()
		defer r.putSkippedNodes(skippedNodes)

		if value := root.getValue(path, ps, skippedNodes, false); value.handler != nil {
			if ps != nil && value.params != nil {
				//_ = pathvar.SetVars(ctx, value.params)
				ctx.SetUserValue(ParamsKey, value.params)
				//defer ctx.RemoveUserValue(ParamsKey)
			}
			value.handler(ctx)

			return
		} else if method != fasthttp.MethodConnect && path != "/" {
			// Moved Permanently, request with GET method
			code := fasthttp.StatusMovedPermanently
			if method != fasthttp.MethodGet {
				// Permanent Redirect, request with same method
				code = fasthttp.StatusPermanentRedirect
			}

			if value.tsr && r.RedirectTrailingSlash {
				//_ = ctx.URI().FullURI()
				if len(path) > 1 && path[len(path)-1] == '/' {
					path = path[:len(path)-1]
				} else {
					path = path + "/"
				}
				ctx.Redirect(path, code)
				return
			}
		}
	}

	if r.HandleMethodNotAllowed && len(r.trees) > 0 { // Handle 405
		// According to RFC 7231 section 6.5.5, MUST generate an Allow header field in response
		// containing a list of the target resource's currently supported methods.
		allowed := make([]string, 0, len(r.trees)-1)
		getSkippedNodes := r.getSkippedNodes()
		defer r.putSkippedNodes(getSkippedNodes)
		for _, tree := range r.trees {
			if tree.method == method {
				continue
			}
			if value := tree.root.getValue(path, nil, getSkippedNodes, false); value.handler != nil {
				allowed = append(allowed, tree.method)
			}
		}
		if len(allowed) > 0 {
			allow := strings.Join(allowed, ", ")
			if r.MethodNotAllowed != nil {
				ctx.Response.Header.Set("Allow", allow)
				r.MethodNotAllowed(ctx)
			} else {
				ctx.Error(
					http.StatusText(fasthttp.StatusMethodNotAllowed),
					fasthttp.StatusMethodNotAllowed,
				)
				ctx.Response.Header.Set("Allow", allow)
			}
			return
		}
	}

	// Handle 404
	if r.NotFound != nil {
		r.NotFound(ctx)
	} else {
		ctx.NotFound()
	}
}
