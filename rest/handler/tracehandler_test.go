package handler

import (
	"context"
	"github.com/r27153733/fastgozero/fastext/otel/propagation"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"

	"net"
	"strconv"
	"testing"

	ztrace "github.com/r27153733/fastgozero/core/trace"
	"github.com/r27153733/fastgozero/core/trace/tracetest"
	"github.com/r27153733/fastgozero/rest/chain"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel"
	tcodes "go.opentelemetry.io/otel/codes"

	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

func TestOtelHandler(t *testing.T) {
	ztrace.StartAgent(ztrace.Config{
		Name:     "go-zero-test",
		Endpoint: "http://localhost:14268/api/traces",
		Batcher:  "jaeger",
		Sampler:  1.0,
	})
	defer ztrace.StopAgent()

	for _, test := range []string{"", "bar"} {
		t.Run(test, func(t *testing.T) {
			h := chain.New(TraceHandler("foo", test)).Then(
				func(ctx *fasthttp.RequestCtx) {
					span := trace.SpanFromContext(ctx)
					assert.True(t, span.SpanContext().IsValid())
					assert.True(t, span.IsRecording())
				})
			ln := fasthttputil.NewInmemoryListener()
			s := fasthttp.Server{
				Handler: h,
			}
			go s.Serve(ln) //nolint:errcheck

			c := &fasthttp.HostClient{
				Dial: func(addr string) (net.Conn, error) {
					return ln.Dial()
				},
			}

			err := func(ctx context.Context) error {
				ctx, span := otel.Tracer("httptrace/client").Start(ctx, "test")
				defer span.End()

				req := fasthttp.AcquireRequest()
				resp := fasthttp.AcquireResponse()
				req.Header.SetMethod(fasthttp.MethodGet)
				req.SetRequestURI("http://localhost")
				otel.GetTextMapPropagator().Inject(ctx, propagation.ConvertReq(&req.Header))
				err := c.Do(req, resp)
				assert.Nil(t, err)
				resp.Body()
				return nil
			}(context.Background())

			assert.Nil(t, err)
		})
	}
}

func TestTraceHandler(t *testing.T) {
	me := tracetest.NewInMemoryExporter(t)
	h := chain.New(TraceHandler("foo", "/")).Then(
		func(ctx *fasthttp.RequestCtx) {})
	ln := fasthttputil.NewInmemoryListener()
	s := fasthttp.Server{
		Handler: h,
	}
	go s.Serve(ln) //nolint:errcheck

	client := &fasthttp.HostClient{
		Dial: func(addr string) (net.Conn, error) {
			return ln.Dial()
		},
	}
	err := func(ctx context.Context) error {
		req := fasthttp.AcquireRequest()
		resp := fasthttp.AcquireResponse()
		req.Header.SetMethod(fasthttp.MethodGet)
		req.SetRequestURI("http://localhost")
		otel.GetTextMapPropagator().Inject(ctx, propagation.ConvertReq(&req.Header))
		err := client.Do(req, resp)
		assert.Nil(t, err)
		resp.Body()
		return nil
	}(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, len(me.GetSpans()))
	span := me.GetSpans()[0].Snapshot()
	assert.Equal(t, sdktrace.Status{
		Code: tcodes.Unset,
	}, span.Status())
	assert.Equal(t, 0, len(span.Events()))
	assert.Equal(t, 10, len(span.Attributes())) // ip
}

func TestDontTracingSpan(t *testing.T) {
	ztrace.StartAgent(ztrace.Config{
		Name:     "go-zero-test",
		Endpoint: "http://localhost:14268/api/traces",
		Batcher:  "jaeger",
		Sampler:  1.0,
	})
	defer ztrace.StopAgent()

	for _, test := range []string{"", "bar", "foo"} {
		t.Run(test, func(t *testing.T) {
			h := chain.New(TraceHandler("foo", test, WithTraceIgnorePaths([]string{"bar"}))).Then(
				func(ctx *fasthttp.RequestCtx) {
					span := trace.SpanFromContext(ctx)
					spanCtx := span.SpanContext()
					if test == "bar" {
						assert.False(t, spanCtx.IsValid())
						assert.False(t, span.IsRecording())
						return
					}

					assert.True(t, span.IsRecording())
					assert.True(t, spanCtx.IsValid())
				})
			ln := fasthttputil.NewInmemoryListener()
			s := fasthttp.Server{
				Handler: h,
			}
			go s.Serve(ln) //nolint:errcheck

			client := &fasthttp.HostClient{
				Dial: func(addr string) (net.Conn, error) {
					return ln.Dial()
				},
			}
			err := func(ctx context.Context) error {
				ctx, span := otel.Tracer("httptrace/client").Start(ctx, "test")
				defer span.End()

				req := fasthttp.AcquireRequest()
				resp := fasthttp.AcquireResponse()
				req.Header.SetMethod(fasthttp.MethodGet)
				req.SetRequestURI("http://localhost")
				otel.GetTextMapPropagator().Inject(ctx, propagation.ConvertReq(&req.Header))
				err := client.Do(req, resp)
				assert.Nil(t, err)
				resp.Body()
				return nil
			}(context.Background())

			assert.Nil(t, err)
		})
	}
}

func TestTraceResponseWriter(t *testing.T) {
	ztrace.StartAgent(ztrace.Config{
		Name:     "go-zero-test",
		Endpoint: "http://localhost:14268/api/traces",
		Batcher:  "jaeger",
		Sampler:  1.0,
	})
	defer ztrace.StopAgent()

	for _, test := range []int{0, 200, 300, 400, 401, 500, 503} {
		t.Run(strconv.Itoa(test), func(t *testing.T) {
			h := chain.New(TraceHandler("foo", "bar")).Then(
				func(ctx *fasthttp.RequestCtx) {
					span := trace.SpanFromContext(ctx)
					spanCtx := span.SpanContext()
					assert.True(t, span.IsRecording())
					assert.True(t, spanCtx.IsValid())
					if test != 0 {
						ctx.SetStatusCode(test)
					}
					ctx.Response.AppendBody([]byte("hello"))
				})
			ln := fasthttputil.NewInmemoryListener()
			s := fasthttp.Server{
				Handler: h,
			}
			go s.Serve(ln) //nolint:errcheck

			client := &fasthttp.HostClient{
				Dial: func(addr string) (net.Conn, error) {
					return ln.Dial()
				},
			}
			err := func(ctx context.Context) error {
				ctx, span := otel.Tracer("httptrace/client").Start(ctx, "test")
				defer span.End()

				req := fasthttp.AcquireRequest()
				resp := fasthttp.AcquireResponse()
				req.Header.SetMethod(fasthttp.MethodGet)
				req.SetRequestURI("http://localhost")
				otel.GetTextMapPropagator().Inject(ctx, propagation.ConvertReq(&req.Header))
				err := client.Do(req, resp)
				assert.Nil(t, err)

				resBody := resp.Body()
				assert.Equal(t, []byte("hello"), resBody, "response body fail")
				return nil
			}(context.Background())

			assert.Nil(t, err)
		})
	}
}
