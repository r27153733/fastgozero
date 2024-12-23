package internal

import (
	"bytes"
	"fmt"
	"github.com/valyala/fasthttp"
	"golang.org/x/net/context"
	"sync"

	"github.com/r27153733/fastgozero/core/logx"
	"github.com/r27153733/fastgozero/rest/httpx"
)

// logContextKey is a context key.
var logContextKey = contextKey("request_logs")

type (
	// LogCollector is used to collect logs.
	LogCollector struct {
		Messages []string
		lock     sync.Mutex
	}

	contextKey string
)

// SetLogCollector returns a new context with LogCollector.
func SetLogCollector(ctx *fasthttp.RequestCtx, lc *LogCollector) (free func()) {
	// logContextKey is private.
	// so we can guarantee that the key will not be overwritten.
	ctx.SetUserValue(logContextKey, lc)
	return func() {
		ctx.RemoveUserValue(logContextKey)
	}
}

// LogCollectorFromContext returns LogCollector from ctx.
func LogCollectorFromContext(ctx context.Context) *LogCollector {
	val := ctx.Value(logContextKey)
	if val == nil {
		return nil
	}

	return val.(*LogCollector)
}

// Append appends msg into log context.
func (lc *LogCollector) Append(msg string) {
	lc.lock.Lock()
	lc.Messages = append(lc.Messages, msg)
	lc.lock.Unlock()
}

// Flush flushes collected logs.
func (lc *LogCollector) Flush() string {
	var buffer bytes.Buffer

	start := true
	for _, message := range lc.takeAll() {
		if start {
			start = false
		} else {
			buffer.WriteByte('\n')
		}
		buffer.WriteString(message)
	}

	return buffer.String()
}

func (lc *LogCollector) takeAll() []string {
	lc.lock.Lock()
	messages := lc.Messages
	lc.Messages = nil
	lc.lock.Unlock()

	return messages
}

// Error logs the given v along with r in error log.
func Error(r *fasthttp.RequestCtx, v ...any) {
	logx.WithContext(r).Error(format(r, v...))
}

// Errorf logs the given v with format along with r in error log.
func Errorf(r *fasthttp.RequestCtx, format string, v ...any) {
	logx.WithContext(r).Error(formatf(r, format, v...))
}

// Info logs the given v along with r in access log.
func Info(r *fasthttp.RequestCtx, v ...any) {
	appendLog(r, format(r, v...))
}

// Infof logs the given v with format along with r in access log.
func Infof(r *fasthttp.RequestCtx, format string, v ...any) {
	appendLog(r, formatf(r, format, v...))
}

func appendLog(r *fasthttp.RequestCtx, message string) {
	logs := LogCollectorFromContext(r)
	if logs != nil {
		logs.Append(message)
	}
}

func format(r *fasthttp.RequestCtx, v ...any) string {
	return formatWithReq(r, fmt.Sprint(v...))
}

func formatf(r *fasthttp.RequestCtx, format string, v ...any) string {
	return formatWithReq(r, fmt.Sprintf(format, v...))
}

func formatWithReq(r *fasthttp.RequestCtx, v string) string {
	return fmt.Sprintf("(%s - %s) %s", r.RequestURI(), httpx.GetRemoteAddr(r), v)
}
