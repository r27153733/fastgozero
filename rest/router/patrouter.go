//go:build gozerorouter

package router

import (
	"errors"
	"net/http"
	"path"
	"strings"

	"github.com/r27153733/fastgozero/core/search"
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/r27153733/fastgozero/rest/router/pathvar"
	"github.com/valyala/fasthttp"
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
	trees      []*search.Tree
	notFound   fasthttp.RequestHandler
	notAllowed fasthttp.RequestHandler
}

// NewRouter returns a httpx.Router.
func NewRouter() httpx.Router {
	return &patRouter{
		trees: make([]*search.Tree, 9),
	}
}

func (pr *patRouter) Handle(method, reqPath string, handler fasthttp.RequestHandler) error {
	if !validMethod(method) {
		return ErrInvalidMethod
	}

	indexOf := methodIndexOf(method)

	if len(reqPath) == 0 || reqPath[0] != '/' {
		return ErrInvalidPath
	}

	cleanPath := path.Clean(reqPath)
	tree := pr.trees[indexOf]
	if tree != nil {
		return tree.Add(cleanPath, handler)
	}

	tree = search.NewTree()
	pr.trees[indexOf] = tree
	return tree.Add(cleanPath, handler)
}

func (pr *patRouter) ServeHTTP(ctx *fasthttp.RequestCtx) {
	reqPath := path.Clean(bytesconv.BToS(ctx.Request.URI().Path()))
	if tree := pr.trees[methodIndexOf(bytesconv.BToS(ctx.Method()))]; tree != nil {
		if result, ok := tree.Search(reqPath); ok {
			if len(result.Params) > 0 {
				free := pathvar.SetVars(ctx, pathvar.MapParams(result.Params))
				defer free()
			}
			result.Item.(fasthttp.RequestHandler)(ctx)
			return
		}
	}

	allows, ok := pr.methodsAllowed(bytesconv.BToS(ctx.Method()), reqPath)
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

	idx := methodIndexOf(method)
	for treeMethod, tree := range pr.trees {
		if treeMethod == idx || tree == nil {
			continue
		}

		_, ok := tree.Search(path)
		if ok {
			allows = append(allows, methodFromIndex(treeMethod))
		}
	}

	if len(allows) > 0 {
		return strings.Join(allows, allowMethodSeparator), true
	}

	return "", false
}

func validMethod(method string) bool {
	return method == fasthttp.MethodDelete || method == fasthttp.MethodGet ||
		method == fasthttp.MethodHead || method == fasthttp.MethodOptions ||
		method == fasthttp.MethodPatch || method == fasthttp.MethodPost ||
		method == fasthttp.MethodPut
}

func methodFromIndex(index int) string {
	switch index {
	case 0:
		return fasthttp.MethodGet
	case 1:
		return fasthttp.MethodHead
	case 2:
		return fasthttp.MethodPost
	case 3:
		return fasthttp.MethodPut
	case 4:
		return fasthttp.MethodPatch
	case 5:
		return fasthttp.MethodDelete
	case 6:
		return fasthttp.MethodConnect
	case 7:
		return fasthttp.MethodOptions
	case 8:
		return fasthttp.MethodTrace
	default:
		return ""
	}
}

func methodIndexOf(method string) int {
	switch method {
	case fasthttp.MethodGet:
		return 0
	case fasthttp.MethodHead:
		return 1
	case fasthttp.MethodPost:
		return 2
	case fasthttp.MethodPut:
		return 3
	case fasthttp.MethodPatch:
		return 4
	case fasthttp.MethodDelete:
		return 5
	case fasthttp.MethodConnect:
		return 6
	case fasthttp.MethodOptions:
		return 7
	case fasthttp.MethodTrace:
		return 8
	default:
		return -1
	}
}
