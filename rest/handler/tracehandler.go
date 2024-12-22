package handler

import (
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"github.com/r27153733/fastgozero/fastext/otel/otelkey"
	"strings"

	"github.com/r27153733/fastgozero/core/collection"
	"github.com/r27153733/fastgozero/core/trace"
	"github.com/r27153733/fastgozero/fastext/otel/propagation"
	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/valyala/fasthttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

type (
	// TraceOption defines the method to customize an traceOptions.
	TraceOption func(options *traceOptions)

	// traceOptions is TraceHandler options.
	traceOptions struct {
		traceIgnorePaths []string
	}
)

var (
	defaultTracerCtxKeys = []any{otelkey.CurrentSpanKey, otelkey.BaggageKey}
)

// TraceHandler return a middleware that process the opentelemetry.
func TraceHandler(serviceName, path string, opts ...TraceOption) func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
	var options traceOptions
	for _, opt := range opts {
		opt(&options)
	}

	ignorePaths := collection.NewSet()
	ignorePaths.AddStr(options.traceIgnorePaths...)

	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		tracer := otel.Tracer(trace.TraceName)
		propagator := otel.GetTextMapPropagator()

		return func(ctx *fasthttp.RequestCtx) {
			spanName := path
			if len(spanName) == 0 {
				spanName = string(ctx.URI().Path())
			}

			if ignorePaths.Contains(spanName) {
				next(ctx)
				return
			}

			addr := httpx.GetRemoteAddr(ctx)
			if i := strings.Index(addr, ","); i > 0 {
				addr = addr[:i]
			}

			attrs := []attribute.KeyValue{
				{Key: "http.target", Value: attribute.StringValue(string(ctx.RequestURI()))},
				{Key: "http.client_ip", Value: attribute.StringValue(addr)},
				{Key: "http.method", Value: attribute.StringValue(string(ctx.Method()))},
			}
			if serviceName != "" {
				attrs = append(attrs, attribute.String("http.server_name", serviceName))
			}
			if spanName != "" {
				attrs = append(attrs, attribute.String("http.route", spanName))
			}
			ua := string(ctx.UserAgent())
			if ua != "" {
				attrs = append(attrs, attribute.String("http.user_agent", ua))
			}
			if contentLength := ctx.Request.Header.ContentLength(); contentLength > 0 {
				attrs = append(attrs, attribute.Int64("http.request_content_length", int64(contentLength)))
			}
			if ctx.IsTLS() {
				attrs = append(attrs, attribute.String("http.scheme", "https"))
			} else {
				attrs = append(attrs, attribute.String("http.scheme", "http"))
			}
			if len(ctx.Request.Host()) != 0 {
				attrs = append(attrs, attribute.String("http.host", string(ctx.Request.Host())))
			} else if ctx.Request.URI() != nil && len(ctx.Request.URI().Host()) != 0 {
				attrs = append(attrs, attribute.String("http.host", string(ctx.Request.URI().Host())))
			}
			flavor := ""
			if bytesconv.BToS(ctx.Request.Header.Protocol()) == "HTTP/2" {
				flavor = "2"
			} else {
				flavor = "1.1"
			}
			attrs = append(attrs, attribute.String("http.flavor", flavor))

			tmp := propagator.Extract(ctx, propagation.ConvertReq(&ctx.Request.Header))
			tmp, span := tracer.Start(
				ctx,
				spanName,
				oteltrace.WithSpanKind(oteltrace.SpanKindServer),
				oteltrace.WithAttributes(attrs...),
			)
			defer span.End()
			for _, key := range defaultTracerCtxKeys {
				// there isn't any user-defined middleware before TraceHandler,
				// so we can guarantee that the key will not be overwritten.
				ctx.SetUserValue(key, tmp.Value(key))
			}
			defer func() {
				for _, key := range defaultTracerCtxKeys {
					ctx.RemoveUserValue(key)
				}
			}()

			// convenient for tracking error messages
			propagator.Inject(ctx, propagation.ConvertResp(&ctx.Response.Header))

			next(ctx)

			code := ctx.Response.StatusCode()
			span.SetAttributes(semconv.HTTPAttributesFromHTTPStatusCode(code)...)
			span.SetStatus(semconv.SpanStatusFromHTTPStatusCodeAndSpanKind(
				code, oteltrace.SpanKindServer))
		}
	}
}

// WithTraceIgnorePaths specifies the traceIgnorePaths option for TraceHandler.
func WithTraceIgnorePaths(traceIgnorePaths []string) TraceOption {
	return func(options *traceOptions) {
		options.traceIgnorePaths = append(options.traceIgnorePaths, traceIgnorePaths...)
	}
}
