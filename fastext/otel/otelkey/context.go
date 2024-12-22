package otelkey

import (
	"context"
	"unsafe"

	"go.opentelemetry.io/otel/baggage"
	"go.opentelemetry.io/otel/trace"
)

var (
	BaggageKey     any
	CurrentSpanKey any
)

func init() {
	ctx := context.Background()

	tmp := baggage.ContextWithoutBaggage(ctx)
	keyValue := getCtxKeyValue(tmp)
	BaggageKey = keyValue.key

	tmp = trace.ContextWithSpan(ctx, nil)
	keyValue = getCtxKeyValue(tmp)
	CurrentSpanKey = keyValue.key
}

type valueCtx struct {
	context.Context
	key, val any
}

type iface struct {
	itab, data unsafe.Pointer
}

func getCtxKeyValue(ctx context.Context) *valueCtx {
	ictx := *(*iface)(unsafe.Pointer(&ctx))
	valCtx := (*valueCtx)(ictx.data)
	return valCtx
}
