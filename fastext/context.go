package fastext

import (
	"context"
	"unsafe"

	"github.com/valyala/fasthttp"
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
	BaggageKey = keyValue

	tmp = trace.ContextWithSpan(ctx, nil)
	keyValue = getCtxKeyValue(tmp)
	CurrentSpanKey = keyValue
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

func SetUserValueCtx(ctx *fasthttp.RequestCtx, key, val any) (free func()) {
	value := ctx.UserValue(key)
	if value == nil {
		free = func() {
			ctx.RemoveUserValue(key)
		}
	} else {
		free = func() {
			ctx.SetUserValue(key, value)
		}
	}
	ctx.SetUserValue(key, val)
	return free
}
