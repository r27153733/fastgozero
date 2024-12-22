package router

import (
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/r27153733/fastgozero/rest/router/pathvar"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

const contentLength = "Content-Length"

type mockedResponseWriter struct {
	code int
}

func (m *mockedResponseWriter) Header() http.Header {
	return http.Header{}
}

func (m *mockedResponseWriter) Write(p []byte) (int, error) {
	return len(p), nil
}

func (m *mockedResponseWriter) WriteHeader(code int) {
	m.code = code
}

func TestPatRouterHandleErrors(t *testing.T) {
	tests := []struct {
		method string
		path   string
		err    error
	}{
		{"FAKE", "", ErrInvalidMethod},
		{"GET", "", ErrInvalidPath},
	}

	for _, test := range tests {
		t.Run(test.method, func(t *testing.T) {
			router := NewRouter()
			err := router.Handle(test.method, test.path, nil)
			assert.Equal(t, test.err, err)
		})
	}
}

func TestPatRouterNotFound(t *testing.T) {
	var notFound bool
	router := NewRouter()
	router.SetNotFoundHandler(func(ctx *fasthttp.RequestCtx) {
		notFound = true
		ctx.SetStatusCode(fasthttp.StatusNotFound)
	})

	err := router.Handle(http.MethodGet, "/a/b", func(ctx *fasthttp.RequestCtx) {})
	if err != nil {
		t.Fatalf("error registering route: %v", err)
	}

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(http.MethodGet)
	ctx.Request.SetRequestURI("/b/c")
	router.ServeHTTP(ctx)

	if !notFound {
		t.Error("expected notFound handler to be triggered, but it was not")
	}
}

func TestPatRouterNotAllowed(t *testing.T) {
	var notAllowed bool
	router := NewRouter()

	// Set a handler for "method not allowed" responses
	router.SetNotAllowedHandler(func(ctx *fasthttp.RequestCtx) {
		notAllowed = true
		ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	})

	// Register a route with the GET method
	err := router.Handle(http.MethodGet, "/a/b", func(ctx *fasthttp.RequestCtx) {})
	if err != nil {
		t.Fatalf("error registering route: %v", err)
	}

	// Create a fasthttp.RequestCtx for a POST request to the same route
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(http.MethodPost) // Set a different method
	ctx.Request.SetRequestURI("/a/b")

	// Pass the request through the router
	router.ServeHTTP(ctx)

	// Check if the "method not allowed" handler was triggered
	if !notAllowed {
		t.Error("expected MethodNotAllowed handler to be triggered, but it was not")
	}
}

func TestPatRouter(t *testing.T) {
	tests := []struct {
		method string
		path   string
		expect bool
		code   int
		err    error
	}{
		// we don't explicitly set status code, framework will do it.
		{http.MethodGet, "/a/b", true, 0, nil},
		{http.MethodGet, "/a/b/", true, 0, nil},
		{http.MethodGet, "/a/b?a=b", true, 0, nil},
		{http.MethodGet, "/a/b/?a=b", true, 0, nil},
		{http.MethodGet, "/a/b/c?a=b", true, 0, nil},
		{http.MethodGet, "/b/d", false, fasthttp.StatusNotFound, nil},
	}

	for _, test := range tests {
		t.Run(test.method+":"+test.path, func(t *testing.T) {
			routed := false
			router := NewRouter()
			// Register routes
			err := router.Handle(test.method, "/a/:b", func(ctx *fasthttp.RequestCtx) {
				routed = true
				assert.Equal(t, 1, pathvar.Vars(ctx).Len())
			})
			assert.Nil(t, err)
			err = router.Handle(test.method, "/a/b/c", func(ctx *fasthttp.RequestCtx) {
				routed = true
				assert.Equal(t, 0, pathvar.Vars(ctx).Len())
			})
			assert.Nil(t, err)
			err = router.Handle(test.method, "/b/c", func(ctx *fasthttp.RequestCtx) {
				routed = true
			})
			assert.Nil(t, err)

			// Simulate a request
			ctx := new(fasthttp.RequestCtx)
			ctx.Request.Header.SetMethod(test.method)
			ctx.Request.SetRequestURI(test.path)
			router.ServeHTTP(ctx)

			if ctx.Response.StatusCode() == fasthttp.StatusMovedPermanently {
				path := string(ctx.Response.Header.Peek("Location"))
				ctx.Response.Reset()
				ctx.Request.SetRequestURI(path)
				router.ServeHTTP(ctx)
			}

			// Assert routing expectations
			assert.Equal(t, test.expect, routed)
			if test.code != 0 {
				assert.Equal(t, test.code, ctx.Response.StatusCode())
			}

			// Test for "method not allowed"
			if test.code == 0 {
				ctx.Request.Header.SetMethod(http.MethodPut)
				ctx.Response.Reset()
				router.ServeHTTP(ctx)
				assert.Equal(t, fasthttp.StatusMethodNotAllowed, ctx.Response.StatusCode())
			}
		})
	}
}

func TestParseSlice(t *testing.T) {
	body := `names=first&names=second`

	// Simulate a request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.SetRequestURI("/")
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.Header.SetContentType("application/x-www-form-urlencoded")
	ctx.Request.SetBody([]byte(body))
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Names []string `form:"names"`
		}{}

		// Parse form values into struct
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err)
		assert.Equal(t, 2, len(v.Names))
		assert.Equal(t, "first", v.Names[0])
		assert.Equal(t, "second", v.Names[1])
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)
}

func TestParseJsonPost(t *testing.T) {
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{"location": "shanghai", "time": 20170912}`))
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err)

		// Write Response
		response := fmt.Sprintf("%s:%d:%s:%d:%s:%d", v.Name, v.Year, v.Nickname, v.Zipcode, v.Location, v.Time)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	router.ServeHTTP(ctx)

	// Assert the response
	assert.Equal(t, "kevin:2017:whatever:200000:shanghai:20170912", string(ctx.Response.Body()))
}

func TestParseJsonPostWithIntSlice(t *testing.T) {
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{"ages": [1, 2], "years": [3, 4]}`))
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name  string  `path:"name"`
			Year  int     `path:"year"`
			Ages  []int   `json:"ages"`
			Years []int64 `json:"years"`
		}{}

		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err)
		assert.ElementsMatch(t, []int{1, 2}, v.Ages)
		assert.ElementsMatch(t, []int64{3, 4}, v.Years)
	})
	assert.Nil(t, err)

	router.ServeHTTP(ctx)
}

func TestParseJsonPostError(t *testing.T) {
	payload := `[{"abcd": "cdef"}]`

	// Simulate a request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(payload))
	ctx.Request.Header.SetContentLength(len(payload))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		// Attempt to parse the request
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // We expect an error here due to invalid JSON payload
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseJsonPostInvalidRequest(t *testing.T) {
	payload := `{"ages": ["cdef"]}`

	// Simulate a request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("/")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(payload))
	ctx.Request.Header.SetContentLength(len(payload))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Ages []int `json:"ages"`
		}{}

		// Attempt to parse the request
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // Expect an error because "cdef" cannot be converted to an integer
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseJsonPostRequired(t *testing.T) {
	// Simulate a request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017")
	ctx.Request.Header.SetContentType("application/json")
	// JSON payload is missing the required `time` field
	ctx.Request.SetBody([]byte(`{"location": "shanghai"`))
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Location string `json:"location"`
			Time     int64  `json:"time"` // Required field
		}{}

		// Attempt to parse the request
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // Expect an error because the `time` field is missing
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParsePath(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017")

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name string `path:"name"`
			Year int    `path:"year"`
		}{}

		// Parse path parameters
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err)

		// Write response
		response := fmt.Sprintf("%s in %d", v.Name, v.Year)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin in 2017", string(ctx.Response.Body()))
}

func TestParsePathRequired(t *testing.T) {
	// Simulate a GET request with fasthttp (missing the "year" parameter in the path)
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin")

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name string `path:"name"`
			Year int    `path:"year"` // This field is required
		}{}

		// Attempt to parse path parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // Expect an error because "year" is missing
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	if ctx.Response.StatusCode() == fasthttp.StatusMovedPermanently {
		path := string(ctx.Response.Header.Peek("Location"))
		ctx.Response.Reset()
		ctx.Request.SetRequestURI(path)
		router.ServeHTTP(ctx)
	}

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseQuery(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000")

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
		}{}

		// Parse query parameters
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err)

		// Write response
		response := fmt.Sprintf("%s:%d", v.Nickname, v.Zipcode)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "whatever:200000", string(ctx.Response.Body()))
}

func TestParseQueryRequired(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever") // Missing `zipcode` query parameter

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"` // Required field
		}{}

		// Attempt to parse query parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // Expect an error because `zipcode` is missing
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseOptional(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever") // Missing `zipcode` (optional field)

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode,optional"` // Optional field
		}{}

		// Parse query parameters
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed even if `zipcode` is missing

		// Write response
		response := fmt.Sprintf("%s:%d", v.Nickname, v.Zipcode)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "whatever:0", string(ctx.Response.Body()))
}

func TestParseNestedInRequestEmpty(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017") // URL with path parameters
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte("{}")) // Empty JSON body
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	type (
		Request struct {
			Name string `path:"name"`
			Year int    `path:"year"`
		}

		Audio struct {
			Volume int `json:"volume"`
		}

		WrappedRequest struct {
			Request
			Audio Audio `json:"audio,optional"` // Optional nested field
		}
	)

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		var v WrappedRequest

		// Parse request
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed, even with an empty JSON body

		// Write response
		response := fmt.Sprintf("%s:%d", v.Name, v.Year)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin:2017", string(ctx.Response.Body()))
}

func TestParsePtrInRequest(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017") // URL with path parameters
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{"audio": {"volume": 100}}`)) // JSON body with audio
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	type (
		Request struct {
			Name string `path:"name"`
			Year int    `path:"year"`
		}

		Audio struct {
			Volume int `json:"volume"`
		}

		WrappedRequest struct {
			Request
			Audio *Audio `json:"audio,optional"` // Optional pointer to a nested struct
		}
	)

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		var v WrappedRequest

		// Parse request
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed

		// Write response
		if v.Audio != nil {
			response := fmt.Sprintf("%s:%d:%d", v.Name, v.Year, v.Audio.Volume)
			ctx.SetBodyString(response)
		} else {
			t.Fatal("audio struct should not be nil")
		}
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin:2017:100", string(ctx.Response.Body()))
}

func TestParsePtrInRequestEmpty(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin") // Path without dynamic parameters
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte("{}")) // Empty JSON body
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	type (
		Audio struct {
			Volume int `json:"volume"`
		}

		WrappedRequest struct {
			Audio *Audio `json:"audio,optional"` // Optional nested field
		}
	)

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/kevin", func(ctx *fasthttp.RequestCtx) {
		var v WrappedRequest

		// Parse the request
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed even with an empty body

		// Assert that the `Audio` pointer is nil since it's optional and not provided in the body
		assert.Nil(t, v.Audio)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate that no error occurred during parsing
	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
}

func TestParseQueryOptional(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=") // Optional `zipcode`

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode,optional"` // Optional field
		}{}

		// Parse query parameters
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed even if `zipcode` is empty

		// Write response
		response := fmt.Sprintf("%s:%d", v.Nickname, v.Zipcode)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "whatever:0", string(ctx.Response.Body()))
}

func TestParse(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000") // Path and query parameters

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
		}{}

		// Parse path and query parameters
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed

		// Write response
		response := fmt.Sprintf("%s:%d:%s:%d", v.Name, v.Year, v.Nickname, v.Zipcode)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin:2017:whatever:200000", string(ctx.Response.Body()))
}

func TestParseWrappedRequest(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017") // Path with dynamic parameters

	type (
		Request struct {
			Name string `path:"name"`
			Year int    `path:"year"`
		}

		WrappedRequest struct {
			Request
		}
	)

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		var v WrappedRequest

		// Parse request
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed

		// Write response
		response := fmt.Sprintf("%s:%d", v.Name, v.Year)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin:2017", string(ctx.Response.Body()))
}

func TestParseWrappedGetRequestWithJsonHeader(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017") // Path with dynamic parameters
	ctx.Request.Header.SetContentType("application/json")    // JSON Content-Type header

	type (
		Request struct {
			Name string `path:"name"`
			Year int    `path:"year"`
		}

		WrappedRequest struct {
			Request
		}
	)

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		var v WrappedRequest

		// Parse request
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed

		// Write response
		response := fmt.Sprintf("%s:%d", v.Name, v.Year)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin:2017", string(ctx.Response.Body()))
}

func TestParseWrappedHeadRequestWithJsonHeader(t *testing.T) {
	// Simulate a HEAD request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodHead)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017") // Path with dynamic parameters
	ctx.Request.Header.SetContentType("application/json")    // JSON Content-Type header

	type (
		Request struct {
			Name string `path:"name"`
			Year int    `path:"year"`
		}

		WrappedRequest struct {
			Request
		}
	)

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodHead, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		var v WrappedRequest

		// Parse request
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed

		// Write response
		response := fmt.Sprintf("%s:%d", v.Name, v.Year)
		ctx.Response.Header.SetContentLength(len(response)) // Set Content-Length for HEAD response
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate that the response doesn't include the body (as it's a HEAD request)
	assert.Equal(t, 0, len(ctx.Response.Body())) // Response body should be empty for HEAD
	assert.Equal(t, len("kevin:2017"), ctx.Response.Header.ContentLength())
}

func TestParseWrappedRequestPtr(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017") // Path with dynamic parameters

	type (
		Request struct {
			Name string `path:"name"`
			Year int    `path:"year"`
		}

		WrappedRequest struct {
			*Request
		}
	)

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		var v WrappedRequest

		// Parse the request
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed

		// Write response
		response := fmt.Sprintf("%s:%d", v.Name, v.Year)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin:2017", string(ctx.Response.Body()))
}

func TestParseWithAll(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000") // Path and query parameters
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{"location": "shanghai", "time": 20170912}`)) // JSON body
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed

		// Write response
		response := fmt.Sprintf("%s:%d:%s:%d:%s:%d", v.Name, v.Year, v.Nickname, v.Zipcode, v.Location, v.Time)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin:2017:whatever:200000:shanghai:20170912", string(ctx.Response.Body()))
}

func TestParseWithAllUtf8(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000") // Path and query parameters
	ctx.Request.Header.SetContentType("application/json; charset=utf-8")                      // JSON Content-Type with UTF-8
	ctx.Request.SetBody([]byte(`{"location": "shanghai", "time": 20170912}`))                 // JSON body
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.Nil(t, err) // Parsing should succeed

		// Write response
		response := fmt.Sprintf("%s:%d:%s:%d:%s:%d", v.Name, v.Year, v.Nickname, v.Zipcode, v.Location, v.Time)
		ctx.SetBodyString(response)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, "kevin:2017:whatever:200000:shanghai:20170912", string(ctx.Response.Body()))
}

func TestParseWithMissingForm(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever") // Missing `zipcode` query parameter
	ctx.Request.Header.SetContentType("application/json")                      // JSON Content-Type
	ctx.Request.SetBody([]byte(`{"location": "shanghai", "time": 20170912}`))  // JSON body
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"` // This field is missing
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // Parsing should fail due to missing `zipcode`
		assert.Equal(t, `field "zipcode" is not set`, err.Error())

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseWithMissingAllForms(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017")                  // Missing all form query parameters
	ctx.Request.Header.SetContentType("application/json")                     // JSON Content-Type
	ctx.Request.SetBody([]byte(`{"location": "shanghai", "time": 20170912}`)) // JSON body
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"` // Missing
			Zipcode  int64  `form:"zipcode"`  // Missing
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err)

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseWithMissingJson(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000") // Path and query parameters
	ctx.Request.Header.SetContentType("application/json")                                     // JSON Content-Type
	ctx.Request.SetBody([]byte(`{"location": "shanghai"}`))                                   // JSON body with missing "time"
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"`
			Time     int64  `json:"time"` // Missing in JSON
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err)                   // Parsing should fail due to missing `time`
		assert.NotEqual(t, io.EOF, err)         // Ensure the error is not due to EOF
		assert.Contains(t, err.Error(), `time`) // Validate that the error mentions the missing `time` field

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseWithMissingAllJsons(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000") // Path and query parameters
	ctx.Request.Header.SetContentType("application/json")                                     // JSON Content-Type
	ctx.Request.SetBody([]byte{})                                                             // Missing JSON body
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"` // Missing in the body
			Time     int64  `json:"time"`     // Missing in the body
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err)           // Parsing should fail due to missing `location` and `time`
		assert.NotEqual(t, io.EOF, err) // Ensure the error is not due to EOF

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseWithMissingPath(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/2017?nickname=whatever&zipcode=200000") // Missing `name` in path
	ctx.Request.Header.SetContentType("application/json")                               // JSON Content-Type
	ctx.Request.SetBody([]byte(`{"location": "shanghai", "time": 20170912}`))           // JSON body
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err)                                 // Parsing should fail due to missing `name`
		assert.Equal(t, "field name is not set", err.Error()) // Validate the error message

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusNotFound, ctx.Response.StatusCode())
}

func TestParseWithMissingAllPaths(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/?nickname=whatever&zipcode=200000") // Missing all path parameters
	ctx.Request.Header.SetContentType("application/json")                           // JSON Content-Type
	ctx.Request.SetBody([]byte(`{"location": "shanghai", "time": 20170912}`))       // JSON body

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"`
			Time     int64  `json:"time"`
		}{}

		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err)

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusNotFound, ctx.Response.StatusCode())
}

func TestParseGetWithContentLengthHeader(t *testing.T) {
	// Simulate a GET request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000") // Path and query parameters
	ctx.Request.Header.SetContentType("application/json")                                     // JSON Content-Type
	ctx.Request.Header.Set("Content-Length", "1024")                                          // Content-Length header

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodGet, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Location string `json:"location"` // Expected in JSON body but not present
			Time     int64  `json:"time"`     // Expected in JSON body but not present
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // Parsing should fail due to missing `location` and `time`

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseJsonPostWithTypeMismatch(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017?nickname=whatever&zipcode=200000") // Path and query parameters
	ctx.Request.Header.SetContentType("application/json")                                     // JSON Content-Type
	ctx.Request.SetBody([]byte(`{"time": "20170912"}`))                                       // JSON body with type mismatch
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name     string `path:"name"`
			Year     int    `path:"year"`
			Nickname string `form:"nickname"`
			Zipcode  int64  `form:"zipcode"`
			Time     int64  `json:"time"` // Expected to be int64, but a string is provided
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // Parsing should fail due to type mismatch

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func TestParseJsonPostWithInt2String(t *testing.T) {
	// Simulate a POST request with fasthttp
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodPost)
	ctx.Request.SetRequestURI("http://hello.com/kevin/2017") // Path parameters
	ctx.Request.Header.SetContentType("application/json")    // JSON Content-Type
	ctx.Request.SetBody([]byte(`{"time": 20170912}`))        // JSON body with an int for a string field
	ctx.Request.Header.SetContentLength(len(ctx.Request.Body()))

	// Define the router
	router := NewRouter()
	err := router.Handle(fasthttp.MethodPost, "/:name/:year", func(ctx *fasthttp.RequestCtx) {
		v := struct {
			Name string `path:"name"`
			Year int    `path:"year"`
			Time string `json:"time"` // Expects a string, but an int is provided
		}{}

		// Parse path, query, and body parameters
		err := httpx.Parse(ctx, &v)
		assert.NotNil(t, err) // Parsing should fail due to type mismatch

		// Respond with an error status
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
	})
	assert.Nil(t, err)

	// Serve the request
	router.ServeHTTP(ctx)

	// Validate the response
	assert.Equal(t, fasthttp.StatusBadRequest, ctx.Response.StatusCode())
}

func BenchmarkPatRouter(b *testing.B) {
	// Create the router
	router := NewRouter()
	_ = router.Handle(fasthttp.MethodGet, "/api/:user/:name", func(ctx *fasthttp.RequestCtx) {
		// Handler does nothing for this benchmark
	})

	// Create a fasthttp RequestCtx for the benchmark
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/a/b")
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)

	b.ResetTimer()
	// Run the benchmark
	for i := 0; i < b.N; i++ {
		router.ServeHTTP(ctx)
	}
}
