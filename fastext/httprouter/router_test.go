// Copyright 2013 Julien Schmidt. All rights reserved.
// Use of this source code is governed by a BSD-style license that can be found
// in the LICENSE file.

package httprouter

import (
	"fmt"
	"github.com/valyala/fasthttp"
	"net/http"
	"reflect"
	"testing"
)

func TestParams(t *testing.T) {
	ps := Params{
		Param{"param1", "value1"},
		Param{"param2", "value2"},
		Param{"param3", "value3"},
	}
	for i := range ps {
		if val := ps.ByName(ps[i].Key); val != ps[i].Value {
			t.Errorf("Wrong value for %s: Got %s; Want %s", ps[i].Key, val, ps[i].Value)
		}
	}
	if val := ps.ByName("noKey"); val != "" {
		t.Errorf("Expected empty string for not found key; got: %s", val)
	}
}

func TestRouter(t *testing.T) {
	router := New()

	routed := false
	router.Handle(http.MethodGet, "/user/:name", func(ctx *fasthttp.RequestCtx) {
		ps := ctx.UserValue(ParamsKey).(Params)
		routed = true
		want := Params{Param{"name", "gopher"}}
		if !reflect.DeepEqual(ps, want) {
			t.Fatalf("wrong wildcard values: want %v, got %v", want, ps)
		}
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/user/gopher")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)

	if !routed {
		t.Fatal("routing failed")
	}
}

func TestRouterAPI(t *testing.T) {
	var get, head, options, post, put, patch, delete, handler bool

	router := New()
	router.GET("/GET", func(ctx *fasthttp.RequestCtx) {
		get = true
	})
	router.HEAD("/GET", func(ctx *fasthttp.RequestCtx) {
		head = true
	})
	router.OPTIONS("/GET", func(ctx *fasthttp.RequestCtx) {
		options = true
	})
	router.POST("/POST", func(ctx *fasthttp.RequestCtx) {
		post = true
	})
	router.PUT("/PUT", func(ctx *fasthttp.RequestCtx) {
		put = true
	})
	router.PATCH("/PATCH", func(ctx *fasthttp.RequestCtx) {
		patch = true
	})
	router.DELETE("/DELETE", func(ctx *fasthttp.RequestCtx) {
		delete = true
	})
	router.Handler(http.MethodGet, "/Handler", func(ctx *fasthttp.RequestCtx) {
		handler = true
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/GET")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !get {
		t.Error("routing GET failed")
	}

	ctx.Request.Header.SetMethod(http.MethodHead)
	ctx.Request.SetRequestURI("/GET")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !head {
		t.Error("routing HEAD failed")
	}

	ctx.Request.Header.SetMethod(http.MethodOptions)
	ctx.Request.SetRequestURI("/GET")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !options {
		t.Error("routing OPTIONS failed")
	}

	ctx.Request.Header.SetMethod(http.MethodPost)
	ctx.Request.SetRequestURI("/POST")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !post {
		t.Error("routing POST failed")
	}

	ctx.Request.Header.SetMethod(http.MethodPut)
	ctx.Request.SetRequestURI("/PUT")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !put {
		t.Error("routing PUT failed")
	}

	ctx.Request.Header.SetMethod(http.MethodPatch)
	ctx.Request.SetRequestURI("/PATCH")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !patch {
		t.Error("routing PATCH failed")
	}

	ctx.Request.Header.SetMethod(http.MethodDelete)
	ctx.Request.SetRequestURI("/DELETE")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !delete {
		t.Error("routing DELETE failed")
	}

	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/Handler")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !handler {
		t.Error("routing Handler failed")
	}
}

func TestRouterInvalidInput(t *testing.T) {
	router := New()

	handle := func(_ *fasthttp.RequestCtx) {}

	recv := catchPanic(func() {
		router.Handle("", "/", handle)
	})
	if recv == nil {
		t.Fatal("registering empty method did not panic")
	}

	recv = catchPanic(func() {
		router.GET("", handle)
	})
	if recv == nil {
		t.Fatal("registering empty path did not panic")
	}

	recv = catchPanic(func() {
		router.GET("noSlashRoot", handle)
	})
	if recv == nil {
		t.Fatal("registering path not beginning with '/' did not panic")
	}

	recv = catchPanic(func() {
		router.GET("/", nil)
	})
	if recv == nil {
		t.Fatal("registering nil handler did not panic")
	}
}

func TestRouterChaining(t *testing.T) {
	router1 := New()
	router2 := New()
	router1.NotFound = router2.ServeHTTP

	fooHit := false
	router1.POST("/foo", func(ctx *fasthttp.RequestCtx) {
		fooHit = true
		ctx.SetStatusCode(http.StatusOK)
	})

	barHit := false
	router2.POST("/bar", func(ctx *fasthttp.RequestCtx) {
		barHit = true
		ctx.SetStatusCode(http.StatusOK)
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("/foo")
	router1.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusOK && fooHit) {
		t.Errorf("Regular routing failed with router chaining.")
		t.FailNow()
	}

	ctx.Request.SetRequestURI("/bar")
	router1.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusOK && barHit) {
		t.Errorf("Chained routing failed with router chaining.")
		t.FailNow()
	}

	ctx.Request.SetRequestURI("/qax")
	router1.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusNotFound) {
		t.Errorf("NotFound behavior failed with router chaining.")
		t.FailNow()
	}
}

func BenchmarkAllowed(b *testing.B) {
	handlerFunc := func(_ *fasthttp.RequestCtx) {}

	router := New()
	router.POST("/path", handlerFunc)
	router.GET("/path", handlerFunc)

	b.Run("Global", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = router.allowed("*", http.MethodOptions)
		}
	})
	b.Run("Path", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = router.allowed("/path", http.MethodOptions)
		}
	})
}

func TestRouterOPTIONS(t *testing.T) {
	handlerFunc := func(_ *fasthttp.RequestCtx) {}

	router := New()
	router.POST("/path", handlerFunc)

	// test not allowed
	// * (server)
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodOptions)
	ctx.Request.SetRequestURI("/*")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d", ctx.Response.StatusCode())
	} else if allow := ctx.Response.Header.Peek("Allow"); string(allow) != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + string(allow))
	}

	// path
	ctx.Request.SetRequestURI("/path")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d", ctx.Response.StatusCode())
	} else if allow := ctx.Response.Header.Peek("Allow"); string(allow) != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + string(allow))
	}

	ctx.Request.SetRequestURI("/doesnotexist")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusNotFound) {
		t.Errorf("OPTIONS handling failed: Code=%d", ctx.Response.StatusCode())
	}

	// add another method
	router.GET("/path", handlerFunc)

	// set a global OPTIONS handler
	router.GlobalOPTIONS = func(ctx *fasthttp.RequestCtx) {
		// Adjust status code to 204
		ctx.Response.SetStatusCode(http.StatusNoContent)
	}

	// test again
	// * (server)
	ctx.Request.SetRequestURI("/*")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d", ctx.Response.StatusCode())
	} else if allow := string(ctx.Response.Header.Peek("Allow")); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// path
	ctx.Request.SetRequestURI("/path")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d", ctx.Response.StatusCode())
	} else if allow := string(ctx.Response.Header.Peek("Allow")); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// custom handler
	var custom bool
	router.OPTIONS("/path", func(ctx *fasthttp.RequestCtx) {
		custom = true
	})

	// test again
	// * (server)
	ctx.Request.SetRequestURI("/*")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusNoContent) {
		t.Errorf("OPTIONS handling failed: Code=%d", ctx.Response.StatusCode())
	} else if allow := string(ctx.Response.Header.Peek("Allow")); allow != "GET, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}
	if custom {
		t.Error("custom handler called on *")
	}

	// path
	ctx.Request.SetRequestURI("/path")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusOK) {
		t.Errorf("OPTIONS handling failed: Code=%d", ctx.Response.StatusCode())
	}
	if !custom {
		t.Error("custom handler not called")
	}
}

func TestRouterNotAllowed(t *testing.T) {
	handlerFunc := func(_ *fasthttp.RequestCtx) {}

	router := New()
	router.POST("/path", handlerFunc)

	// test not allowed
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/path")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusMethodNotAllowed) {
		t.Errorf("NotAllowed handling failed: Code=%d", ctx.Response.StatusCode())
	} else if allow := string(ctx.Response.Header.Peek("Allow")); allow != "OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	// add another method
	router.DELETE("/path", handlerFunc)
	router.OPTIONS("/path", handlerFunc) // must be ignored

	// test again
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/path")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusMethodNotAllowed) {
		t.Errorf("NotAllowed handling failed: Code=%d", ctx.Response.StatusCode())
	} else if allow := string(ctx.Response.Header.Peek("Allow")); allow != "DELETE, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}

	responseText := "custom method"
	router.MethodNotAllowed = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(http.StatusTeapot)
		ctx.Response.AppendBodyString(responseText)
	}
	ctx.Response.Reset()
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if got := string(ctx.Response.Body()); !(got == responseText) {
		t.Errorf("unexpected response got %q want %q", got, responseText)
	}
	if ctx.Response.StatusCode() != http.StatusTeapot {
		t.Errorf("unexpected response code %d want %d", ctx.Response.StatusCode(), http.StatusTeapot)
	}
	if allow := string(ctx.Response.Header.Peek("Allow")); allow != "DELETE, OPTIONS, POST" {
		t.Error("unexpected Allow header value: " + allow)
	}
}

func buildLocation(host, path string) string {
	return fmt.Sprintf("http://%s%s", host, path)
}

func TestRouterNotFound(t *testing.T) {
	handlerFunc := func(ctx *fasthttp.RequestCtx) {}
	host := "fast"
	router := New()
	router.GET("/path", handlerFunc)
	router.GET("/dir/", handlerFunc)
	router.GET("/", handlerFunc)

	testRoutes := []struct {
		route    string
		code     int
		location string
	}{
		{"/path/", http.StatusMovedPermanently, buildLocation("fast", "/path")}, // TSR -/
		{"/dir", http.StatusMovedPermanently, buildLocation("fast", "/dir/")},   // TSR +/
		{"", http.StatusOK, ""},                                   // TSR +/
		{"/nope", http.StatusNotFound, buildLocation("fast", "")}, // NotFound
	}

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetHost(host)
	ctx.Request.Header.SetMethod(http.MethodGet)

	for _, tr := range testRoutes {
		ctx.Request.SetRequestURI(tr.route)
		ctx.Response.Reset()
		router.ServeHTTP(ctx)
		if !(ctx.Response.StatusCode() == tr.code && (ctx.Response.StatusCode() == http.StatusNotFound || string(ctx.Response.Header.Peek("Location")) == tr.location)) {
			t.Errorf("NotFound handling route %s failed: Code=%d, Header=%v", tr.route, ctx.Response.StatusCode(), string(ctx.Response.Header.Peek("Location")))
		}
	}

	// Test custom not found handler
	var notFound bool
	router.NotFound = func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(http.StatusNotFound)
		notFound = true
	}
	ctx.Request.SetRequestURI("/nope")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusNotFound && notFound == true) {
		t.Errorf("Custom NotFound handler failed: Code=%d", ctx.Response.StatusCode())
	}

	// Test other method than GET (want 308 instead of 301)
	router.PATCH("/path", handlerFunc)
	ctx.Request.Header.SetMethod(http.MethodPatch)
	ctx.Request.SetRequestURI("/path/")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusPermanentRedirect && string(ctx.Response.Header.Peek("Location")) == buildLocation("fast", "/path")) {
		t.Errorf("Custom NotFound handler failed: Code=%d", ctx.Response.StatusCode())
	}

	// Test special case where no node for the prefix "/" exists
	router = New()
	router.GET("/a", handlerFunc)
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !(ctx.Response.StatusCode() == http.StatusNotFound) {
		t.Errorf("NotFound handling route / failed: Code=%d", ctx.Response.StatusCode())
	}
}

func TestRouterLookup(t *testing.T) {
	routed := false
	wantHandle := func(_ *fasthttp.RequestCtx) {
		routed = true
	}
	wantParams := Params{Param{"name", "gopher"}}

	router := New()

	// try empty router first
	handle, _, tsr := router.Lookup(http.MethodGet, "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}

	// insert route and try again
	router.GET("/user/:name", wantHandle)
	handle, params, _ := router.Lookup(http.MethodGet, "/user/gopher")
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle(nil)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}
	if !reflect.DeepEqual(params, wantParams) {
		t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, params)
	}
	routed = false

	// route without param
	router.GET("/user", wantHandle)
	handle, params, _ = router.Lookup(http.MethodGet, "/user")
	if handle == nil {
		t.Fatal("Got no handle!")
	} else {
		handle(nil)
		if !routed {
			t.Fatal("Routing failed!")
		}
	}
	if params != nil {
		t.Fatalf("Wrong parameter values: want %v, got %v", nil, params)
	}

	handle, _, tsr = router.Lookup(http.MethodGet, "/user/gopher/")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if !tsr {
		t.Error("Got no TSR recommendation!")
	}

	handle, _, tsr = router.Lookup(http.MethodGet, "/nope")
	if handle != nil {
		t.Fatalf("Got handle for unregistered pattern: %v", handle)
	}
	if tsr {
		t.Error("Got wrong TSR recommendation!")
	}
}

func TestRouterParamsFromContext(t *testing.T) {
	routed := false

	wantParams := Params{Param{"name", "gopher"}}
	handlerFunc := func(ctx *fasthttp.RequestCtx) {
		// get params from request context
		params := ParamsFromContext(ctx)

		if !reflect.DeepEqual(params, wantParams) {
			t.Fatalf("Wrong parameter values: want %v, got %v", wantParams, params)
		}

		routed = true
	}

	var nilParams Params
	handlerFuncNil := func(ctx *fasthttp.RequestCtx) {
		// get params from request context
		params := ParamsFromContext(ctx)

		if !reflect.DeepEqual(params, nilParams) {
			t.Fatalf("Wrong parameter values: want %v, got %v", nilParams, params)
		}

		routed = true
	}
	router := New()
	router.Handler(http.MethodGet, "/user", handlerFuncNil)
	router.Handler(http.MethodGet, "/user/:name", handlerFunc)

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/user/gopher")
	ctx.Response.Reset()
	router.ServeHTTP(ctx)
	if !routed {
		t.Fatal("Routing failed!")
	}

	//routed = false
	//ctx.Request.SetRequestURI("/user")
	//ctx.Response.Reset()
	//router.ServeHTTP(ctx)
	//if !routed {
	//	t.Fatal("Routing failed!")
	//}
}
