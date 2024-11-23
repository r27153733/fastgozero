package handler

import (
	"bytes"
	"fmt"
	"strconv"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/core/color"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/syncx"
	"github.com/zeromicro/go-zero/core/timex"
	"github.com/zeromicro/go-zero/core/utils"
	"github.com/zeromicro/go-zero/fastext"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/internal"
)

const (
	limitBodyBytes       = 1024
	defaultSlowThreshold = time.Millisecond * 500
)

var slowThreshold = syncx.ForAtomicDuration(defaultSlowThreshold)

// LogHandler returns a middleware that logs http request and response.
func LogHandler(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		timer := utils.NewElapsedTimer()
		logs := new(internal.LogCollector)
		free := internal.SetLogCollector(ctx, logs)
		defer free()
		next(ctx)

		logBrief(ctx, timer, logs)
	}
}

// DetailedLogHandler returns a middleware that logs http request and response in details.
func DetailedLogHandler(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		timer := utils.NewElapsedTimer()

		logs := new(internal.LogCollector)
		free := internal.SetLogCollector(ctx, logs)
		defer free()
		next(ctx)

		logDetails(ctx, timer, logs)
	}
}

// SetSlowThreshold sets the slow threshold.
func SetSlowThreshold(threshold time.Duration) {
	slowThreshold.Set(threshold)
}

func isOkResponse(code int) bool {
	// not server error
	return code < fasthttp.StatusInternalServerError
}

func logBrief(r *fasthttp.RequestCtx, timer *utils.ElapsedTimer, logs *internal.LogCollector) {
	var buf bytes.Buffer
	code := r.Response.StatusCode()
	duration := timer.Duration()
	logger := logx.WithContext(r).WithDuration(duration)
	buf.WriteString(fmt.Sprintf("[HTTP] %s - %s %s - %s - %s",
		wrapStatusCode(code), wrapMethod(fastext.B2s(r.Method())), fastext.B2s(r.RequestURI()), httpx.GetRemoteAddr(r), fastext.B2s(r.UserAgent())))
	if duration > slowThreshold.Load() {
		logger.Slowf("[HTTP] %s - %s %s - %s - %s - slowcall(%s)",
			wrapStatusCode(code), wrapMethod(fastext.B2s(r.Method())), fastext.B2s(r.RequestURI()), httpx.GetRemoteAddr(r), fastext.B2s(r.UserAgent()),
			timex.ReprOfDuration(duration))
	}

	ok := isOkResponse(code)
	if !ok {
		buf.WriteString(fmt.Sprintf("\n%s", r.Request.String()))
	}

	body := logs.Flush()
	if len(body) > 0 {
		buf.WriteString(fmt.Sprintf("\n%s", body))
	}

	if ok {
		logger.Info(buf.String())
	} else {
		logger.Error(buf.String())
	}
}

func logDetails(ctx *fasthttp.RequestCtx, timer *utils.ElapsedTimer,
	logs *internal.LogCollector) {
	var buf bytes.Buffer
	duration := timer.Duration()
	code := ctx.Response.StatusCode()
	logger := logx.WithContext(ctx)
	buf.WriteString(fmt.Sprintf("[HTTP] %s - %d - %s - %s\n=> %s\n",
		fastext.B2s(ctx.Method()), code, ctx.RemoteAddr().String(), timex.ReprOfDuration(duration), ctx.Request.String()))
	if duration > defaultSlowThreshold {
		logger.Slowf("[HTTP] %s - %d - %s - slowcall(%s)\n=> %s\n", fastext.B2s(ctx.Method()), code, ctx.RemoteAddr().String(),
			fmt.Sprintf("slowcall(%s)", timex.ReprOfDuration(duration)), ctx.Request.String())
	}

	body := logs.Flush()
	if len(body) > 0 {
		buf.WriteString(fmt.Sprintf("%s\n", body))
	}

	respBuf := ctx.Response.String()
	if len(respBuf) > 0 {
		buf.WriteString(fmt.Sprintf("<= %s", respBuf))
	}

	if isOkResponse(code) {
		logger.Info(buf.String())
	} else {
		logger.Error(buf.String())
	}
}

func wrapMethod(method string) string {
	var colour color.Color
	switch method {
	case fasthttp.MethodGet:
		colour = color.BgBlue
	case fasthttp.MethodPost:
		colour = color.BgCyan
	case fasthttp.MethodPut:
		colour = color.BgYellow
	case fasthttp.MethodDelete:
		colour = color.BgRed
	case fasthttp.MethodPatch:
		colour = color.BgGreen
	case fasthttp.MethodHead:
		colour = color.BgMagenta
	case fasthttp.MethodOptions:
		colour = color.BgWhite
	}

	if colour == color.NoColor {
		return method
	}

	return logx.WithColorPadding(method, colour)
}

func wrapStatusCode(code int) string {
	var colour color.Color
	switch {
	case code >= fasthttp.StatusOK && code < fasthttp.StatusMultipleChoices:
		colour = color.BgGreen
	case code >= fasthttp.StatusMultipleChoices && code < fasthttp.StatusBadRequest:
		colour = color.BgBlue
	case code >= fasthttp.StatusBadRequest && code < fasthttp.StatusInternalServerError:
		colour = color.BgMagenta
	default:
		colour = color.BgYellow
	}

	return logx.WithColorPadding(strconv.Itoa(code), colour)
}
