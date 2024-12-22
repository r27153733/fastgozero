// Copyright 2014 Manu Martinez-Almeida. All rights reserved.
// Use of this source code is governed by a MIT style
// license that can be found in the LICENSE file.

package httprouter

import (
	"bytes"
	"github.com/valyala/fasthttp"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type header struct {
	Key   string
	Value string
}

// PerformRequest for testing gin router.
func PerformRequest(r *Router, method, path string, headers ...header) *httptest.ResponseRecorder {
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(path)
	for _, h := range headers {
		ctx.Request.Header.Add(h.Key, h.Value)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(ctx)
	w.Code = ctx.Response.StatusCode()
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		w.Header().Add(string(key), string(value))
	})
	w.Body = bytes.NewBuffer(ctx.Response.Body())
	return w
}

func testRouteOK(method string, t *testing.T) {
	passed := false
	r := New()

	err := r.Handle(method, "/test", func(c *fasthttp.RequestCtx) {
		passed = true
	})
	assert.NoError(t, err)

	w := PerformRequest(r, method, "/test")
	assert.True(t, passed)
	assert.Equal(t, http.StatusOK, w.Code)
}

// TestSingleRouteOK tests that POST route is correctly invoked.
func testRouteNotOK(method string, t *testing.T) {
	passed := false
	router := New()
	err := router.Handle(method, "/test_2", func(c *fasthttp.RequestCtx) {
		passed = true
	})
	assert.NoError(t, err)

	w := PerformRequest(router, method, "/test")

	assert.False(t, passed)
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestSingleRouteOK tests that POST route is correctly invoked.
func testRouteNotOK2(method string, t *testing.T) {
	passed := false
	router := New()
	router.HandleMethodNotAllowed = true
	var methodRoute string
	if method == http.MethodPost {
		methodRoute = http.MethodGet
	} else {
		methodRoute = http.MethodPost
	}
	err := router.Handle(methodRoute, "/test", func(c *fasthttp.RequestCtx) {
		passed = true
	})
	assert.NoError(t, err)

	w := PerformRequest(router, method, "/test")

	assert.False(t, passed)
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestRouterMethod(t *testing.T) {
	router := New()
	router.PUT("/hey2", func(c *fasthttp.RequestCtx) {
		c.Response.SetStatusCode(fasthttp.StatusOK)
		c.Response.SetBodyString("sup2")
	})

	router.PUT("/hey", func(c *fasthttp.RequestCtx) {
		c.Response.SetStatusCode(fasthttp.StatusOK)
		c.Response.SetBodyString("called")
	})

	router.PUT("/hey3", func(c *fasthttp.RequestCtx) {
		c.Response.SetStatusCode(fasthttp.StatusOK)
		c.Response.SetBodyString("sup3")
	})

	w := PerformRequest(router, http.MethodPut, "/hey")

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "called", w.Body.String())
}

func TestRouterGroupRouteOK(t *testing.T) {
	testRouteOK(http.MethodGet, t)
	testRouteOK(http.MethodPost, t)
	testRouteOK(http.MethodPut, t)
	testRouteOK(http.MethodPatch, t)
	testRouteOK(http.MethodHead, t)
	testRouteOK(http.MethodOptions, t)
	testRouteOK(http.MethodDelete, t)
	testRouteOK(http.MethodConnect, t)
	testRouteOK(http.MethodTrace, t)
}

func TestRouteNotOK(t *testing.T) {
	testRouteNotOK(http.MethodGet, t)
	testRouteNotOK(http.MethodPost, t)
	testRouteNotOK(http.MethodPut, t)
	testRouteNotOK(http.MethodPatch, t)
	testRouteNotOK(http.MethodHead, t)
	testRouteNotOK(http.MethodOptions, t)
	testRouteNotOK(http.MethodDelete, t)
	testRouteNotOK(http.MethodConnect, t)
	testRouteNotOK(http.MethodTrace, t)
}

func TestRouteNotOK2(t *testing.T) {
	testRouteNotOK2(http.MethodGet, t)
	testRouteNotOK2(http.MethodPost, t)
	testRouteNotOK2(http.MethodPut, t)
	testRouteNotOK2(http.MethodPatch, t)
	testRouteNotOK2(http.MethodHead, t)
	testRouteNotOK2(http.MethodOptions, t)
	testRouteNotOK2(http.MethodDelete, t)
	testRouteNotOK2(http.MethodConnect, t)
	testRouteNotOK2(http.MethodTrace, t)
}

func TestRouteRedirectTrailingSlash(t *testing.T) {
	router := New()
	router.RedirectTrailingSlash = true
	router.GET("/path", func(c *fasthttp.RequestCtx) {})
	router.GET("/path2/", func(c *fasthttp.RequestCtx) {})
	router.POST("/path3", func(c *fasthttp.RequestCtx) {})
	router.PUT("/path4/", func(c *fasthttp.RequestCtx) {})

	w := PerformRequest(router, http.MethodGet, "/path/")

	assert.Equal(t, "http://"+"/path", w.Header().Get("Location"))
	assert.Equal(t, http.StatusMovedPermanently, w.Code)

	w = PerformRequest(router, http.MethodGet, "/path2")
	assert.Equal(t, "http://"+"/path2/", w.Header().Get("Location"))
	assert.Equal(t, http.StatusMovedPermanently, w.Code)

	w = PerformRequest(router, http.MethodPost, "/path3/")
	assert.Equal(t, "http://"+"/path3", w.Header().Get("Location"))
	assert.Equal(t, http.StatusPermanentRedirect, w.Code)

	w = PerformRequest(router, http.MethodPut, "/path4")
	assert.Equal(t, "http://"+"/path4/", w.Header().Get("Location"))
	assert.Equal(t, http.StatusPermanentRedirect, w.Code)

	w = PerformRequest(router, http.MethodGet, "/path")
	assert.Equal(t, http.StatusOK, w.Code)

	w = PerformRequest(router, http.MethodGet, "/path2/")
	assert.Equal(t, http.StatusOK, w.Code)

	w = PerformRequest(router, http.MethodPost, "/path3")
	assert.Equal(t, http.StatusOK, w.Code)

	w = PerformRequest(router, http.MethodPut, "/path4/")
	assert.Equal(t, http.StatusOK, w.Code)

	//w = PerformRequest(router, http.MethodGet, "/path2", header{Key: "X-Forwarded-Prefix", Value: "/api"})
	//assert.Equal(t, "/api/path2/", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path2/", header{Key: "X-Forwarded-Prefix", Value: "/api/"})
	//assert.Equal(t, http.StatusOK, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path/", header{Key: "X-Forwarded-Prefix", Value: "../../api#?"})
	//assert.Equal(t, "/api/path", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path/", header{Key: "X-Forwarded-Prefix", Value: "../../api"})
	//assert.Equal(t, "/api/path", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path2", header{Key: "X-Forwarded-Prefix", Value: "../../api"})
	//assert.Equal(t, "/api/path2/", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path2", header{Key: "X-Forwarded-Prefix", Value: "/../../api"})
	//assert.Equal(t, "/api/path2/", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path/", header{Key: "X-Forwarded-Prefix", Value: "api/../../"})
	//assert.Equal(t, "//path", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path/", header{Key: "X-Forwarded-Prefix", Value: "api/../../../"})
	//assert.Equal(t, "/path", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path2", header{Key: "X-Forwarded-Prefix", Value: "../../gin-gonic.com"})
	//assert.Equal(t, "/gin-goniccom/path2/", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path2", header{Key: "X-Forwarded-Prefix", Value: "/../../gin-gonic.com"})
	//assert.Equal(t, "/gin-goniccom/path2/", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path/", header{Key: "X-Forwarded-Prefix", Value: "https://gin-gonic.com/#"})
	//assert.Equal(t, "https/gin-goniccom/https/gin-goniccom/path", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path/", header{Key: "X-Forwarded-Prefix", Value: "#api"})
	//assert.Equal(t, "api/api/path", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path/", header{Key: "X-Forwarded-Prefix", Value: "/nor-mal/#?a=1"})
	//assert.Equal(t, "/nor-mal/a1/path", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)
	//
	//w = PerformRequest(router, http.MethodGet, "/path/", header{Key: "X-Forwarded-Prefix", Value: "/nor-mal/%2e%2e/"})
	//assert.Equal(t, "/nor-mal/2e2e/path", w.Header().Get("Location"))
	//assert.Equal(t, http.StatusMovedPermanently, w.Code)

	router.RedirectTrailingSlash = false

	w = PerformRequest(router, http.MethodGet, "/path/")
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = PerformRequest(router, http.MethodGet, "/path2")
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = PerformRequest(router, http.MethodPost, "/path3/")
	assert.Equal(t, http.StatusNotFound, w.Code)
	w = PerformRequest(router, http.MethodPut, "/path4")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

// TestContextParamsGet tests that a parameter can be parsed from the URL.
//func TestRouteParamsByName(t *testing.T) {
//	name := ""
//	lastName := ""
//	wild := ""
//	router := New()
//	router.GET("/test/:name/:last_name/*wild", func(c *fasthttp.RequestCtx) {
//		p := c.Value(ParamsKey).(Params)
//		name = p.ByName("name")
//		lastName = p.ByName("last_name")
//		var ok bool
//		wild, ok = p.Get("wild")
//
//		assert.True(t, ok)
//		assert.Equal(t, name, c.Param("name"))
//		assert.Equal(t, lastName, c.Param("last_name"))
//
//		assert.Empty(t, c.Param("wtf"))
//		assert.Empty(t, c.Params.ByName("wtf"))
//
//		wtf, ok := c.Params.Get("wtf")
//		assert.Empty(t, wtf)
//		assert.False(t, ok)
//	})
//
//	w := PerformRequest(router, http.MethodGet, "/test/john/smith/is/super/great")
//
//	assert.Equal(t, http.StatusOK, w.Code)
//	assert.Equal(t, "john", name)
//	assert.Equal(t, "smith", lastName)
//	assert.Equal(t, "/is/super/great", wild)
//}

// TestRouteParamsNotEmpty tests that context parameters will be set
// even if a route with params/wildcards is registered after the context
// initialisation (which happened in a previous requests).
//func TestRouteParamsNotEmpty(t *testing.T) {
//	name := ""
//	lastName := ""
//	wild := ""
//	router := New()
//
//	w := PerformRequest(router, http.MethodGet, "/test/john/smith/is/super/great")
//
//	assert.Equal(t, http.StatusNotFound, w.Code)
//
//	router.GET("/test/:name/:last_name/*wild", func(c *Context) {
//		name = c.Params.ByName("name")
//		lastName = c.Params.ByName("last_name")
//		var ok bool
//		wild, ok = c.Params.Get("wild")
//
//		assert.True(t, ok)
//		assert.Equal(t, name, c.Param("name"))
//		assert.Equal(t, lastName, c.Param("last_name"))
//
//		assert.Empty(t, c.Param("wtf"))
//		assert.Empty(t, c.Params.ByName("wtf"))
//
//		wtf, ok := c.Params.Get("wtf")
//		assert.Empty(t, wtf)
//		assert.False(t, ok)
//	})
//
//	w = PerformRequest(router, http.MethodGet, "/test/john/smith/is/super/great")
//
//	assert.Equal(t, http.StatusOK, w.Code)
//	assert.Equal(t, "john", name)
//	assert.Equal(t, "smith", lastName)
//	assert.Equal(t, "/is/super/great", wild)
//}

func TestRouteNotAllowedEnabled(t *testing.T) {
	router := New()
	router.HandleMethodNotAllowed = true
	router.POST("/path", func(c *fasthttp.RequestCtx) {})
	w := PerformRequest(router, http.MethodGet, "/path")
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)

	router.SetNotAllowedHandler(func(c *fasthttp.RequestCtx) {
		c.Response.SetStatusCode(fasthttp.StatusTeapot)
		c.Response.SetBodyString("responseText")
	})
	w = PerformRequest(router, http.MethodGet, "/path")
	assert.Equal(t, "responseText", w.Body.String())
	assert.Equal(t, http.StatusTeapot, w.Code)
}

func TestRouteNotAllowedEnabled2(t *testing.T) {
	router := New()
	router.HandleMethodNotAllowed = true
	// add one methodTree to trees
	err := router.Handle(http.MethodPost, "/", func(ctx *fasthttp.RequestCtx) {

	})
	assert.NoError(t, err)

	router.GET("/path2", func(c *fasthttp.RequestCtx) {})
	w := PerformRequest(router, http.MethodPost, "/path2")
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestRouteNotAllowedEnabled3(t *testing.T) {
	router := New()
	router.HandleMethodNotAllowed = true
	router.GET("/path", func(c *fasthttp.RequestCtx) {})
	router.POST("/path", func(c *fasthttp.RequestCtx) {})
	w := PerformRequest(router, http.MethodPut, "/path")
	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	allowed := w.Header().Get("Allow")
	assert.Contains(t, allowed, "GET")
	assert.Contains(t, allowed, "POST")
}

func TestRouteNotAllowedDisabled(t *testing.T) {
	router := New()
	router.HandleMethodNotAllowed = false
	router.POST("/path", func(c *fasthttp.RequestCtx) {})
	w := PerformRequest(router, http.MethodGet, "/path")
	assert.Equal(t, http.StatusNotFound, w.Code)

	router.SetNotAllowedHandler(func(c *fasthttp.RequestCtx) {
		c.Response.SetStatusCode(fasthttp.StatusTeapot)
		c.Response.SetBodyString("responseText")
	})
	w = PerformRequest(router, http.MethodGet, "/path")
	assert.Equal(t, "404 Page not found", w.Body.String())
	assert.Equal(t, http.StatusNotFound, w.Code)
}

//func TestRouteContextHoldsFullPath(t *testing.T) {
//	router := New()
//
//	// Test routes
//	routes := []string{
//		"/simple",
//		"/project/:name",
//		"/",
//		"/news/home",
//		"/news",
//		"/simple-two/one",
//		"/simple-two/one-two",
//		"/project/:name/build/*params",
//		"/project/:name/bui",
//		"/user/:id/status",
//		"/user/:id",
//		"/user/:id/profile",
//	}
//
//	for _, route := range routes {
//		actualRoute := route
//		router.GET(route, func(c *fasthttp.RequestCtx) {
//			// For each defined route context should contain its full path
//			assert.Equal(t, actualRoute, c.FullPath())
//			c.AbortWithStatus(http.StatusOK)
//		})
//	}
//
//	for _, route := range routes {
//		w := PerformRequest(router, http.MethodGet, route)
//		assert.Equal(t, http.StatusOK, w.Code)
//	}
//
//	// Test not found
//	router.Use(func(c *Context) {
//		// For not found routes full path is empty
//		assert.Equal(t, "", c.FullPath())
//	})
//
//	w := PerformRequest(router, http.MethodGet, "/not-found")
//	assert.Equal(t, http.StatusNotFound, w.Code)
//}
