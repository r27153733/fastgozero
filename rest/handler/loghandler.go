package handler

import (
	"bufio"
	"strconv"
	"time"

	"github.com/valyala/bytebufferpool"
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

var (
	logBriefBodyPool   bytebufferpool.Pool
	logDetailsBodyPool bytebufferpool.Pool
)

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
	code := r.Response.StatusCode()
	duration := timer.Duration()
	logger := logx.WithContext(r).WithDuration(duration)
	ok := isOkResponse(code)

	var buf *bytebufferpool.ByteBuffer
	if ok {
		buf = logBriefBodyPool.Get()
		defer logBriefBodyPool.Put(buf)
	} else {
		buf = new(bytebufferpool.ByteBuffer)
	}

	buf.B = append(buf.B, "[HTTP] "...)
	buf.B = append(buf.B, wrapStatusCode(code)...)
	buf.B = append(buf.B, " - "...)
	buf.B = append(buf.B, wrapMethod(fastext.B2s(r.Method()))...)
	buf.B = append(buf.B, " "...)
	buf.B = append(buf.B, r.RequestURI()...)
	buf.B = append(buf.B, " - "...)
	buf.B = append(buf.B, httpx.GetRemoteAddr(r)...)
	buf.B = append(buf.B, " - "...)
	buf.B = append(buf.B, r.UserAgent()...)

	if duration > slowThreshold.Load() {
		logger.Slowf("%s - slowcall(%s)", fastext.B2s(buf.B), timex.ReprOfDuration(duration))
	}

	if !ok {
		buf.B = append(buf.B, '\n')
		err := r.Request.Write(bufio.NewWriterSize(buf, 1))
		if err != nil {
			panic("BUG!" + err.Error())
		}
	}

	body := logs.Flush()
	if len(body) > 0 {
		buf.B = append(buf.B, '\n')
		buf.B = append(buf.B, body...)
	}

	if ok {
		logger.Info(buf.String())
	} else {
		logger.Error(buf.String())
	}
}

func logDetails(ctx *fasthttp.RequestCtx, timer *utils.ElapsedTimer,
	logs *internal.LogCollector) {
	buf := logDetailsBodyPool.Get()
	defer logDetailsBodyPool.Put(buf)

	duration := timer.Duration()
	code := ctx.Response.StatusCode()
	logger := logx.WithContext(ctx)

	buf.B = append(buf.B, "[HTTP] "...)
	buf.B = append(buf.B, ctx.Method()...)
	buf.B = append(buf.B, " - "...)
	buf.B = append(buf.B, wrapStatusCode(code)...)
	buf.B = append(buf.B, " - "...)
	buf.B = append(buf.B, ctx.RemoteAddr().String()...)
	buf.B = append(buf.B, " - "...)
	if duration > defaultSlowThreshold {
		l := len(buf.B)
		timeStr := timex.ReprOfDuration(duration)

		buf.B = append(buf.B, "slowcall("...)
		buf.B = append(buf.B, timeStr...)
		buf.B = append(buf.B, ')')
		buf.B = append(buf.B, "\n=> "...)
		_, err := ctx.Request.WriteTo(buf)
		if err != nil {
			panic("BUG!" + err.Error())
		}
		buf.B = append(buf.B, '\n')

		logger.Slow(fastext.B2s(buf.B))

		copy(buf.B[l:], timeStr)
		l = l + len(timeStr)
		buf.B = append(buf.B[:l], buf.B[l+10:]...)
	} else {
		buf.B = append(buf.B, timex.ReprOfDuration(duration)...)
		buf.B = append(buf.B, "\n=> "...)
		_, err := ctx.Request.WriteTo(buf)
		if err != nil {
			panic("BUG!" + err.Error())
		}
		buf.B = append(buf.B, '\n')
	}

	body := logs.Flush()
	if len(body) > 0 {
		buf.B = append(buf.B, body...)
		buf.B = append(buf.B, '\n')
	}

	buf.B = append(buf.B, "<= "...)
	_, err := ctx.Response.WriteTo(buf)
	if err != nil {
		panic("BUG!" + err.Error())
	}
	buf.B = append(buf.B, '\n')

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
