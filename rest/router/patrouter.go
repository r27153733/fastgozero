package router

import (
	"errors"
	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/fastext"
	"net/http"
	"path"
	"strings"

	"github.com/zeromicro/go-zero/core/search"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"
)

const (
	allowHeader          = "Allow"
	allowMethodSeparator = ", "
)

var (
	// ErrInvalidMethod is an error that indicates not a valid http method.
	ErrInvalidMethod = errors.New("not a valid http method")
	// ErrInvalidPath is an error that indicates path is not start with /.
	ErrInvalidPath = errors.New("path must begin with '/'")
)

type patRouter struct {
	trees      map[string]*search.Tree
	notFound   fasthttp.RequestHandler
	notAllowed fasthttp.RequestHandler
}

// NewRouter returns a httpx.Router.
func NewRouter() httpx.Router {
	return &patRouter{
		trees: make(map[string]*search.Tree),
	}
}

func (pr *patRouter) Handle(method, reqPath string, handler fasthttp.RequestHandler) error {
	if !validMethod(method) {
		return ErrInvalidMethod
	}

	if len(reqPath) == 0 || reqPath[0] != '/' {
		return ErrInvalidPath
	}

	cleanPath := path.Clean(reqPath)
	tree, ok := pr.trees[method]
	if ok {
		return tree.Add(cleanPath, handler)
	}

	tree = search.NewTree()
	pr.trees[method] = tree
	return tree.Add(cleanPath, handler)
}

func (pr *patRouter) ServeHTTP(ctx *fasthttp.RequestCtx) {
	reqPath := path.Clean(fastext.B2s(ctx.Request.URI().Path()))
	if tree, ok := pr.trees[fastext.B2s(ctx.Method())]; ok {
		if result, ok := tree.Search(reqPath); ok {
			if len(result.Params) > 0 {
				free := pathvar.SetVars(ctx, result.Params)
				defer free()
			}
			result.Item.(fasthttp.RequestHandler)(ctx)
			return
		}
	}

	allows, ok := pr.methodsAllowed(fastext.B2s(ctx.Method()), reqPath)
	if !ok {
		pr.handleNotFound(ctx)
		return
	}

	if pr.notAllowed != nil {
		pr.notAllowed(ctx)
	} else {
		ctx.Response.Header.Set(allowHeader, allows)
		ctx.SetStatusCode(http.StatusMethodNotAllowed)
	}
}

func (pr *patRouter) SetNotFoundHandler(handler fasthttp.RequestHandler) {
	pr.notFound = handler
}

func (pr *patRouter) SetNotAllowedHandler(handler fasthttp.RequestHandler) {
	pr.notAllowed = handler
}

func (pr *patRouter) handleNotFound(ctx *fasthttp.RequestCtx) {
	if pr.notFound != nil {
		pr.notFound(ctx)
	} else {
		ctx.NotFound()
	}
}

func (pr *patRouter) methodsAllowed(method, path string) (string, bool) {
	var allows []string

	for treeMethod, tree := range pr.trees {
		if treeMethod == method {
			continue
		}

		_, ok := tree.Search(path)
		if ok {
			allows = append(allows, treeMethod)
		}
	}

	if len(allows) > 0 {
		return strings.Join(allows, allowMethodSeparator), true
	}

	return "", false
}

func validMethod(method string) bool {
	return method == http.MethodDelete || method == http.MethodGet ||
		method == http.MethodHead || method == http.MethodOptions ||
		method == http.MethodPatch || method == http.MethodPost ||
		method == http.MethodPut
}
