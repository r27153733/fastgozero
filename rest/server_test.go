package rest

import (
	"crypto/tls"
	"embed"
	"fmt"
	"github.com/r27153733/fastgozero/core/conf"
	"github.com/r27153733/fastgozero/core/logx/logtest"
	"github.com/r27153733/fastgozero/rest/chain"
	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/r27153733/fastgozero/rest/internal/cors"
	"github.com/r27153733/fastgozero/rest/router"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"io"
	"io/fs"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

const (
	exampleContent = "example content"
	sampleContent  = "sample content"
)

func TestNewServer(t *testing.T) {
	logtest.Discard(t)

	const configYaml = `
Name: foo
Host: localhost
Port: 0
`
	var cnf RestConf
	assert.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))

	tests := []struct {
		c    RestConf
		opts []RunOption
		fail bool
	}{
		{
			c:    RestConf{},
			opts: []RunOption{WithRouter(mockedRouter{}), WithCors()},
		},
		{
			c:    cnf,
			opts: []RunOption{WithRouter(mockedRouter{})},
		},
		{
			c:    cnf,
			opts: []RunOption{WithRouter(mockedRouter{}), WithNotAllowedHandler(nil)},
		},
		{
			c:    cnf,
			opts: []RunOption{WithNotFoundHandler(nil), WithRouter(mockedRouter{})},
		},
		{
			c:    cnf,
			opts: []RunOption{WithUnauthorizedCallback(nil), WithRouter(mockedRouter{})},
		},
		{
			c:    cnf,
			opts: []RunOption{WithUnsignedCallback(nil), WithRouter(mockedRouter{})},
		},
	}

	for _, test := range tests {
		var svr *Server
		var err error
		if test.fail {
			_, err = NewServer(test.c, test.opts...)
			assert.NotNil(t, err)
			continue
		} else {
			svr = MustNewServer(test.c, test.opts...)
		}

		svr.Use(ToMiddleware(func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return func(r *fasthttp.RequestCtx) {
				next(r)
			}
		}))
		svr.AddRoute(Route{
			Method:  http.MethodGet,
			Path:    "/",
			Handler: nil,
		}, WithJwt("thesecret"), WithSignature(SignatureConf{}),
			WithJwtTransition("preivous", "thenewone"))

		func() {
			defer func() {
				p := recover()
				switch v := p.(type) {
				case error:
					assert.Equal(t, "foo", v.Error())
				default:
					t.Fail()
				}
			}()

			svr.Start()
			svr.Stop()
		}()

		func() {
			defer func() {
				p := recover()
				switch v := p.(type) {
				case error:
					assert.Equal(t, "foo", v.Error())
				default:
					t.Fail()
				}
			}()

			svr.StartWithOpts(func(svr *fasthttp.Server) {

			})
			svr.Stop()
		}()
	}
}

func TestWithMaxBytes(t *testing.T) {
	const maxBytes = 1000
	var fr featuredRoutes
	WithMaxBytes(maxBytes)(&fr)
	assert.Equal(t, int64(maxBytes), fr.maxBytes)
}

func TestWithMiddleware(t *testing.T) {
	m := make(map[string]string)
	rt := router.NewRouter()
	handler := func(r *fasthttp.RequestCtx) {
		var v struct {
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
		}

		err := httpx.Parse(r, &v)
		assert.Nil(t, err)
		_, err = io.WriteString(r.Response.BodyWriter(), fmt.Sprintf("%s:%d", v.Nickname, v.Zipcode))
		assert.Nil(t, err)
	}
	rs := WithMiddleware(func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(r *fasthttp.RequestCtx) {
			var v struct {
				Name string `path:"name"`
				Year string `path:"year"`
			}
			assert.Nil(t, httpx.ParsePath(r, &v))
			m[v.Name] = v.Year
			next(r)
		}
	}, Route{
		Method:  http.MethodGet,
		Path:    "/first/:name/:year",
		Handler: handler,
	}, Route{
		Method:  http.MethodGet,
		Path:    "/second/:name/:year",
		Handler: handler,
	})

	urls := []string{
		"http://hello.com/first/kevin/2017?nickname=whatever&zipcode=200000",
		"http://hello.com/second/wan/2020?nickname=whatever&zipcode=200000",
	}
	for _, route := range rs {
		assert.Nil(t, rt.Handle(route.Method, route.Path, route.Handler))
	}
	for _, url := range urls {
		r := new(fasthttp.RequestCtx)
		r.Request.Header.SetMethod(fasthttp.MethodGet)
		r.Request.SetRequestURI(url)

		rt.ServeHTTP(r)

		assert.Equal(t, "whatever:200000", string(r.Response.Body()))
	}

	assert.EqualValues(t, map[string]string{
		"kevin": "2017",
		"wan":   "2020",
	}, m)
}

func TestWithFileServerMiddleware(t *testing.T) {
	tests := []struct {
		name            string
		path            string
		dir             string
		requestPath     string
		expectedStatus  int
		expectedContent string
	}{
		{
			name:            "Serve static file",
			path:            "/assets/",
			dir:             "./testdata",
			requestPath:     "/assets/example.txt",
			expectedStatus:  http.StatusOK,
			expectedContent: exampleContent,
		},
		{
			name:           "Pass through non-matching path",
			path:           "/static/",
			dir:            "./testdata",
			requestPath:    "/other/path",
			expectedStatus: http.StatusNotFound,
		},
		{
			name:            "Directory with trailing slash",
			path:            "/static",
			dir:             "testdata",
			requestPath:     "/static/sample.txt",
			expectedStatus:  http.StatusOK,
			expectedContent: sampleContent,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := MustNewServer(RestConf{}, WithFileServer(tt.path, os.DirFS(tt.dir)))
			r := new(fasthttp.RequestCtx)
			r.Request.Header.SetMethod(fasthttp.MethodGet)
			r.Request.SetRequestURI(tt.requestPath)

			server.ServeHTTP(r)

			assert.Equal(t, tt.expectedStatus, r.Response.StatusCode())
			if len(tt.expectedContent) > 0 {
				assert.Equal(t, tt.expectedContent, string(r.Response.Body()))
			}
		})
	}
}

func TestMultiMiddlewares(t *testing.T) {
	m := make(map[string]string)
	rt := router.NewRouter()
	handler := func(r *fasthttp.RequestCtx) {
		var v struct {
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
		}

		err := httpx.Parse(r, &v)
		assert.Nil(t, err)
		_, err = io.WriteString(r.Response.BodyWriter(), fmt.Sprintf("%s:%s", v.Nickname, m[v.Nickname]))
		assert.Nil(t, err)
	}
	rs := WithMiddlewares([]Middleware{
		func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return func(r *fasthttp.RequestCtx) {
				var v struct {
					Name string `path:"name"`
					Year string `path:"year"`
				}
				assert.Nil(t, httpx.ParsePath(r, &v))
				m[v.Name] = v.Year
				next(r)
			}
		},
		func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return func(r *fasthttp.RequestCtx) {
				var v struct {
					Name    string `form:"nickname"`
					Zipcode string `form:"zipcode"`
				}
				assert.Nil(t, httpx.ParseForm(&r.Request, &v))
				assert.NotEmpty(t, m)
				m[v.Name] = v.Zipcode + v.Zipcode
				next(r)
			}
		},
		ToMiddleware(func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return next
		}),
	}, Route{
		Method:  http.MethodGet,
		Path:    "/first/:name/:year",
		Handler: handler,
	}, Route{
		Method:  http.MethodGet,
		Path:    "/second/:name/:year",
		Handler: handler,
	})

	urls := []string{
		"http://hello.com/first/kevin/2017?nickname=whatever&zipcode=200000",
		"http://hello.com/second/wan/2020?nickname=whatever&zipcode=200000",
	}
	for _, route := range rs {
		assert.Nil(t, rt.Handle(route.Method, route.Path, route.Handler))
	}
	for _, url := range urls {
		r := new(fasthttp.RequestCtx)
		r.Request.Header.SetMethod(fasthttp.MethodGet)
		r.Request.SetRequestURI(url)

		rt.ServeHTTP(r)

		assert.Equal(t, "whatever:200000200000", string(r.Response.Body()))
	}

	assert.EqualValues(t, map[string]string{
		"kevin":    "2017",
		"wan":      "2020",
		"whatever": "200000200000",
	}, m)
}

func TestWithPrefix(t *testing.T) {
	fr := featuredRoutes{
		routes: []Route{
			{
				Path: "/hello",
			},
			{
				Path: "/world",
			},
		},
	}
	WithPrefix("/api")(&fr)
	vals := make([]string, 0, len(fr.routes))
	for _, r := range fr.routes {
		vals = append(vals, r.Path)
	}
	assert.EqualValues(t, []string{"/api/hello", "/api/world"}, vals)
}

func TestWithPriority(t *testing.T) {
	var fr featuredRoutes
	WithPriority()(&fr)
	assert.True(t, fr.priority)
}

func TestWithTimeout(t *testing.T) {
	var fr featuredRoutes
	WithTimeout(time.Hour)(&fr)
	assert.Equal(t, time.Hour, fr.timeout)
}

func TestWithTLSConfig(t *testing.T) {
	const configYaml = `
Name: foo
Port: 54321
`
	var cnf RestConf
	assert.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))

	testConfig := &tls.Config{
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		},
	}

	testCases := []struct {
		c    RestConf
		opts []RunOption
		res  *tls.Config
	}{
		{
			c:    cnf,
			opts: []RunOption{WithTLSConfig(testConfig)},
			res:  testConfig,
		},
		{
			c:    cnf,
			opts: []RunOption{WithUnsignedCallback(nil)},
			res:  nil,
		},
	}

	for _, testCase := range testCases {
		svr, err := NewServer(testCase.c, testCase.opts...)
		assert.Nil(t, err)
		assert.Equal(t, svr.ngin.tlsConfig, testCase.res)
	}
}

func TestWithCors(t *testing.T) {
	const configYaml = `
Name: foo
Port: 54321
`
	var cnf RestConf
	assert.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))
	rt := router.NewRouter()
	svr, err := NewServer(cnf, WithRouter(rt))
	assert.Nil(t, err)
	defer svr.Stop()

	opt := WithCors("local")
	opt(svr)
}

func TestWithCustomCors(t *testing.T) {
	const configYaml = `
Name: foo
Port: 54321
`
	var cnf RestConf
	assert.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))
	rt := router.NewRouter()
	svr, err := NewServer(cnf, WithRouter(rt))
	assert.Nil(t, err)

	opt := WithCustomCors(func(header *fasthttp.ResponseHeader) {
		header.Set("foo", "bar")
	}, func(w *fasthttp.Response) {
		w.SetStatusCode(http.StatusOK)
	}, "local")
	opt(svr)
}

func TestWithCorsHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
	}{
		{
			name:    "single header",
			headers: []string{"UserHeader"},
		},
		{
			name:    "multiple headers",
			headers: []string{"UserHeader", "X-Requested-With"},
		},
		{
			name:    "no headers",
			headers: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const configYaml = `
Name: foo
Port: 54321
`
			var cnf RestConf
			assert.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))
			rt := router.NewRouter()
			svr, err := NewServer(cnf, WithRouter(rt))
			assert.Nil(t, err)
			defer svr.Stop()
			option := WithCorsHeaders(tt.headers...)
			option(svr)

			// Assuming newCorsRouter sets headers correctly,
			// we would need to verify the behavior here. Since we don't have
			// direct access to headers, we'll mock newCorsRouter to capture it.
			r := new(fasthttp.RequestCtx)
			r.Request.Header.SetMethod(fasthttp.MethodOptions)
			r.Request.SetRequestURI("?")
			svr.ServeHTTP(r)

			vals := r.Response.Header.PeekAll("Access-Control-Allow-Headers")
			respHeaders := make(map[string]struct{})
			for _, header := range vals {
				headers := strings.Split(string(header), ", ")
				for _, h := range headers {
					if len(h) > 0 {
						respHeaders[h] = struct{}{}
					}
				}
			}
			for _, h := range tt.headers {
				_, ok := respHeaders[h]
				assert.Truef(t, ok, "expected header %s not found", h)
			}
		})
	}
}

func TestServer_PrintRoutes(t *testing.T) {
	const (
		configYaml = `
Name: foo
Port: 54321
`
		expect = `Routes:
  GET /bar
  GET /foo
  GET /foo/:bar
  GET /foo/:bar/baz
`
	)

	var cnf RestConf
	assert.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))

	svr, err := NewServer(cnf)
	assert.Nil(t, err)
	notFound := func(ctx *fasthttp.RequestCtx) {
		ctx.NotFound()
	}
	svr.AddRoutes([]Route{
		{
			Method:  fasthttp.MethodGet,
			Path:    "/foo",
			Handler: notFound,
		},
		{
			Method:  fasthttp.MethodGet,
			Path:    "/bar",
			Handler: notFound,
		},
		{
			Method:  fasthttp.MethodGet,
			Path:    "/foo/:bar",
			Handler: notFound,
		},
		{
			Method:  fasthttp.MethodGet,
			Path:    "/foo/:bar/baz",
			Handler: notFound,
		},
	})

	old := os.Stdout
	r, w, err := os.Pipe()
	assert.Nil(t, err)
	os.Stdout = w
	defer func() {
		os.Stdout = old
	}()

	svr.PrintRoutes()
	ch := make(chan string)

	go func() {
		var buf strings.Builder
		io.Copy(&buf, r)
		ch <- buf.String()
	}()

	w.Close()
	out := <-ch
	assert.Equal(t, expect, out)
}

func TestServer_Routes(t *testing.T) {
	const (
		configYaml = `
Name: foo
Port: 54321
`
		expect = `GET /foo GET /bar GET /foo/:bar GET /foo/:bar/baz`
	)

	var cnf RestConf
	assert.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))

	svr, err := NewServer(cnf)
	assert.Nil(t, err)
	notFound := func(ctx *fasthttp.RequestCtx) {
		ctx.NotFound()
	}
	svr.AddRoutes([]Route{
		{
			Method:  http.MethodGet,
			Path:    "/foo",
			Handler: notFound,
		},
		{
			Method:  http.MethodGet,
			Path:    "/bar",
			Handler: notFound,
		},
		{
			Method:  http.MethodGet,
			Path:    "/foo/:bar",
			Handler: notFound,
		},
		{
			Method:  http.MethodGet,
			Path:    "/foo/:bar/baz",
			Handler: notFound,
		},
	})

	routes := svr.Routes()
	var buf strings.Builder
	for i := 0; i < len(routes); i++ {
		buf.WriteString(routes[i].Method)
		buf.WriteString(" ")
		buf.WriteString(routes[i].Path)
		buf.WriteString(" ")
	}

	assert.Equal(t, expect, strings.Trim(buf.String(), " "))
}

func TestHandleError(t *testing.T) {
	assert.NotPanics(t, func() {
		handleError(nil)
		handleError(http.ErrServerClosed)
	})
}

func TestValidateSecret(t *testing.T) {
	assert.Panics(t, func() {
		validateSecret("short")
	})
}

func TestServer_WithChain(t *testing.T) {
	var called int32
	middleware1 := func() func(fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return func(r *fasthttp.RequestCtx) {
				atomic.AddInt32(&called, 1)
				next(r)
				atomic.AddInt32(&called, 1)
			}
		}
	}
	middleware2 := func() func(fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
			return func(r *fasthttp.RequestCtx) {
				atomic.AddInt32(&called, 1)
				next(r)
				atomic.AddInt32(&called, 1)
			}
		}
	}

	server := MustNewServer(RestConf{}, WithChain(chain.New(middleware1(), middleware2())))
	server.AddRoutes(
		[]Route{
			{
				Method: http.MethodGet,
				Path:   "/",
				Handler: func(_ *fasthttp.RequestCtx) {
					atomic.AddInt32(&called, 1)
				},
			},
		},
	)
	rt := router.NewRouter()
	assert.Nil(t, server.ngin.bindRoutes(rt))
	r := new(fasthttp.RequestCtx)
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	r.Request.SetRequestURI("/")
	rt.ServeHTTP(r)
	assert.Equal(t, int32(5), atomic.LoadInt32(&called))
}

func TestServer_WithCors(t *testing.T) {
	var called int32
	middleware := func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(r *fasthttp.RequestCtx) {
			atomic.AddInt32(&called, 1)
			next(r)
		}
	}
	r := router.NewRouter()
	assert.Nil(t, r.Handle(http.MethodOptions, "/", middleware(func(ctx *fasthttp.RequestCtx) {
		ctx.NotFound()
	})))

	cr := &corsRouter{
		Router:     r,
		middleware: cors.Middleware(nil, "*"),
	}
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodOptions)
	req.Request.SetRequestURI("/")
	cr.ServeHTTP(req)
	assert.Equal(t, int32(0), atomic.LoadInt32(&called))
}

func TestServer_ServeHTTP(t *testing.T) {
	const configYaml = `
Name: foo
Port: 54321
`

	var cnf RestConf
	assert.Nil(t, conf.LoadFromYamlBytes([]byte(configYaml), &cnf))

	svr, err := NewServer(cnf)
	assert.Nil(t, err)

	svr.AddRoutes([]Route{
		{
			Method: http.MethodGet,
			Path:   "/foo",
			Handler: func(ctx *fasthttp.RequestCtx) {
				_, _ = ctx.Response.BodyWriter().Write([]byte("succeed"))
				ctx.Response.SetStatusCode(fasthttp.StatusOK)
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/bar",
			Handler: func(ctx *fasthttp.RequestCtx) {
				_, _ = ctx.Response.BodyWriter().Write([]byte("succeed"))
				ctx.Response.SetStatusCode(fasthttp.StatusOK)
			},
		},
		{
			Method: http.MethodGet,
			Path:   "/user/:name",
			Handler: func(ctx *fasthttp.RequestCtx) {
				var userInfo struct {
					Name string `path:"name"`
				}

				err := httpx.Parse(ctx, &userInfo)
				if err != nil {
					_, _ = ctx.Response.BodyWriter().Write([]byte("failed"))
					ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
					return
				}

				_, _ = ctx.Response.BodyWriter().Write([]byte("succeed"))
				ctx.Response.SetStatusCode(fasthttp.StatusOK)
			},
		},
	})

	testCase := []struct {
		name string
		path string
		code int
	}{
		{
			name: "URI : /foo",
			path: "/foo",
			code: http.StatusOK,
		},
		{
			name: "URI : /bar",
			path: "/bar",
			code: http.StatusOK,
		},
		{
			name: "URI : undefined path",
			path: "/test",
			code: http.StatusNotFound,
		},
		{
			name: "URI : /user/:name",
			path: "/user/abc",
			code: http.StatusOK,
		},
	}

	for _, test := range testCase {
		t.Run(test.name, func(t *testing.T) {
			r := new(fasthttp.RequestCtx)

			r.Request.Header.SetMethod(fasthttp.MethodGet)
			r.Request.SetRequestURI(test.path)
			svr.ServeHTTP(r)
			assert.Equal(t, test.code, r.Response.StatusCode())
		})
	}
}

//go:embed testdata
var content embed.FS

func TestServerEmbedFileSystem(t *testing.T) {
	filesys, err := fs.Sub(content, "testdata")
	assert.NoError(t, err)

	server := MustNewServer(RestConf{}, WithFileServer("/assets", filesys))

	r := new(fasthttp.RequestCtx)
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	r.Request.SetRequestURI("/assets/sample.txt")
	server.ServeHTTP(r)
	assert.Equal(t, sampleContent, string(r.Response.Body()))
}
