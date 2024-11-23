package chain

import (
	"net"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// A constructor for middleware
// that writes its own "tag" into the RW and does nothing else.
// Useful in checking if a chain is behaving in the right order.
func tagMiddleware(tag string) Middleware {
	return func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			ctx.Response.AppendBodyString(tag)
			h(ctx)
		}
	}
}

// Not recommended (https://golang.org/pkg/reflect/#Value.Pointer),
// but the best we can do.
func funcsEqual(f1, f2 any) bool {
	val1 := reflect.ValueOf(f1)
	val2 := reflect.ValueOf(f2)
	return val1.Pointer() == val2.Pointer()
}

var testApp = func(ctx *fasthttp.RequestCtx) {
	ctx.Response.AppendBody([]byte("app\n"))
}

// StripPrefix returns a handler that serves HTTP requests by removing the
// given prefix from the request URL's Path (and RawPath if set) and invoking
// the handler h. StripPrefix handles a request for a path that doesn't begin
// with prefix by replying with an HTTP 404 not found error. The prefix must
// match exactly: if the prefix in the request contains escaped characters
// the reply is also an HTTP 404 not found error.
func StripPrefix(prefix string, h fasthttp.RequestHandler) fasthttp.RequestHandler {
	if prefix == "" {
		return h
	}
	return func(ctx *fasthttp.RequestCtx) {
		sPath := string(ctx.Request.URI().Path())
		p := strings.TrimPrefix(sPath, prefix)
		if len(p) < len(sPath) {
			u := fasthttp.AcquireURI()

			u.SetPath(p)
			ctx.Request.SetURI(u)
			//r2.URL = new(url.URL)
			//*r2.URL = *r.URL
			//r2.URL.Path = p
			//r2.URL.RawPath = rp
			//h.ServeHTTP(w, r2)
		} else {
			ctx.NotFound()
		}
	}
}

func TestNew(t *testing.T) {
	c1 := func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return nil
	}

	c2 := func(h fasthttp.RequestHandler) fasthttp.RequestHandler {
		return StripPrefix("potato", nil)
	}

	slice := []Middleware{c1, c2}
	c := New(slice...)
	for k := range slice {
		assert.True(t, funcsEqual(c.(chain).middlewares[k], slice[k]),
			"New does not add constructors correctly")
	}
}

func TestThenWorksWithNoMiddleware(t *testing.T) {
	assert.True(t, funcsEqual(New().Then(testApp), testApp),
		"Then does not work with no middleware")
}

func TestThenTreatsNilAsDefaultServeMux(t *testing.T) {
	assert.True(t, funcsEqual(DefaultServeMux, New().Then(nil)),
		"Then does not treat nil as DefaultServeMux")
}

func TestThenFuncTreatsNilAsDefaultServeMux(t *testing.T) {
	assert.True(t, funcsEqual(DefaultServeMux, New().ThenFunc(nil)),
		"Then does not treat nil as DefaultServeMux")
}

func TestThenFuncConstructsHandlerFunc(t *testing.T) {
	fn := func(ctx *fasthttp.RequestCtx) {
		ctx.ResetBody()
		ctx.Response.SetStatusCode(200)
	}
	chained := New().ThenFunc(fn)

	assert.Equal(t, reflect.TypeOf((fasthttp.RequestHandler)(nil)), reflect.TypeOf(chained),
		"ThenFunc does not construct HandlerFunc")
}

func TestThenOrdersHandlersCorrectly(t *testing.T) {
	t1 := tagMiddleware("t1\n")
	t2 := tagMiddleware("t2\n")
	t3 := tagMiddleware("t3\n")

	chained := New(t1, t2, t3).Then(testApp)

	ln := fasthttputil.NewInmemoryListener()
	s := &fasthttp.Server{
		Handler: chained,
	}
	go s.Serve(ln) //nolint:errcheck
	c := &fasthttp.HostClient{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://example.com")
	res := fasthttp.AcquireResponse()
	err := c.Do(req, res)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, []byte("t1\nt2\nt3\napp\n"), res.Body(),
		"Then does not order handlers correctly")
}

func TestAppendAddsHandlersCorrectly(t *testing.T) {
	c := New(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	c = c.Append(tagMiddleware("t3\n"), tagMiddleware("t4\n"))
	h := c.Then(testApp)

	ln := fasthttputil.NewInmemoryListener()
	s := &fasthttp.Server{
		Handler: h,
	}
	go s.Serve(ln) //nolint:errcheck
	cc := &fasthttp.HostClient{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://example.com")
	res := fasthttp.AcquireResponse()
	err := cc.Do(req, res)
	assert.Nil(t, err)

	assert.Equal(t, []byte("t1\nt2\nt3\nt4\napp\n"), res.Body(),
		"Append does not add handlers correctly")
}

func TestExtendAddsHandlersCorrectly(t *testing.T) {
	c := New(tagMiddleware("t3\n"), tagMiddleware("t4\n"))
	c = c.Prepend(tagMiddleware("t1\n"), tagMiddleware("t2\n"))
	h := c.Then(testApp)

	ln := fasthttputil.NewInmemoryListener()
	s := &fasthttp.Server{
		Handler: h,
	}
	go s.Serve(ln) //nolint:errcheck
	cc := &fasthttp.HostClient{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	req := fasthttp.AcquireRequest()
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI("http://example.com")
	res := fasthttp.AcquireResponse()
	err := cc.Do(req, res)
	assert.Nil(t, err)

	assert.Equal(t, []byte("t1\nt2\nt3\nt4\napp\n"), res.Body(),
		"Extend does not add handlers in correctly")
}
