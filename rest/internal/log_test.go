package internal

import (
	"context"
	"github.com/valyala/fasthttp"
	"strings"
	"testing"

	"github.com/r27153733/fastgozero/core/logx/logtest"
	"github.com/stretchr/testify/assert"
)

func TestInfo(t *testing.T) {
	collector := new(LogCollector)
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://localhost")
	SetLogCollector(ctx, collector)

	Info(ctx, "first")
	Infof(ctx, "second %s", "third")
	val := collector.Flush()
	assert.True(t, strings.Contains(val, "first"))
	assert.True(t, strings.Contains(val, "second"))
	assert.True(t, strings.Contains(val, "third"))
	assert.True(t, strings.Contains(val, "\n"))
}

func TestError(t *testing.T) {
	c := logtest.NewCollector(t)
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)
	ctx.Request.SetRequestURI("http://localhost")
	Error(ctx, "first")
	Errorf(ctx, "second %s", "third")
	val := c.String()
	assert.True(t, strings.Contains(val, "first"))
	assert.True(t, strings.Contains(val, "second"))
	assert.True(t, strings.Contains(val, "third"))
}

func TestLogCollectorContext(t *testing.T) {
	ctx := context.Background()
	assert.Nil(t, LogCollectorFromContext(ctx))
	collector := new(LogCollector)
	ctx = context.WithValue(ctx, logContextKey, collector)
	assert.Equal(t, collector, LogCollectorFromContext(ctx))
}
