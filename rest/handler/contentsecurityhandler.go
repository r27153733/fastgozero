package handler

import (
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"time"

	"github.com/r27153733/fastgozero/core/codec"
	"github.com/r27153733/fastgozero/core/logx"
	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/r27153733/fastgozero/rest/internal/security"
	"github.com/valyala/fasthttp"
)

const contentSecurity = "X-Content-Security"

// UnsignedCallback defines the method of the unsigned callback.
type UnsignedCallback func(ctx *fasthttp.RequestCtx, next fasthttp.RequestHandler, strict bool, code int)

// ContentSecurityHandler returns a middleware to verify content security.
func ContentSecurityHandler(decrypters map[string]codec.RsaDecrypter, tolerance time.Duration,
	strict bool, callbacks ...UnsignedCallback) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	return LimitContentSecurityHandler(maxBytes, decrypters, tolerance, strict, callbacks...)
}

// LimitContentSecurityHandler returns a middleware to verify content security.
func LimitContentSecurityHandler(limitBytes int64, decrypters map[string]codec.RsaDecrypter,
	tolerance time.Duration, strict bool, callbacks ...UnsignedCallback) func(handler fasthttp.RequestHandler) fasthttp.RequestHandler {
	if len(callbacks) == 0 {
		callbacks = append(callbacks, handleVerificationFailure)
	}

	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			switch bytesconv.BToS(ctx.Method()) {
			case fasthttp.MethodDelete, fasthttp.MethodGet, fasthttp.MethodPost, fasthttp.MethodPut:
				header, err := security.ParseContentSecurity(decrypters, bytesconv.BToS(ctx.Request.Header.Peek(httpx.ContentSecurity)))
				if err != nil {
					logx.Errorf("Signature parse failed, X-Content-Security: %s, error: %s",
						bytesconv.BToS(ctx.Request.Header.Peek(contentSecurity)), err.Error())
					executeCallbacks(ctx, next, strict, httpx.CodeSignatureInvalidHeader, callbacks)
				} else if code := security.VerifySignature(&ctx.Request, header, tolerance); code != httpx.CodeSignaturePass {
					logx.Errorf("Signature verification failed, X-Content-Security: %s",
						bytesconv.BToS(ctx.Request.Header.Peek(contentSecurity)))
					executeCallbacks(ctx, next, strict, code, callbacks)
				} else if ctx.Request.Header.ContentLength() > 0 && header.Encrypted() {
					LimitCryptionHandler(limitBytes, header.Key)(next)(ctx)
				} else {
					next(ctx)
				}
			default:
				next(ctx)
			}
		}
	}
}

func executeCallbacks(ctx *fasthttp.RequestCtx, next fasthttp.RequestHandler, strict bool,
	code int, callbacks []UnsignedCallback) {
	for _, callback := range callbacks {
		callback(ctx, next, strict, code)
	}
}

func handleVerificationFailure(ctx *fasthttp.RequestCtx, next fasthttp.RequestHandler,
	strict bool, _ int) {
	if strict {
		ctx.SetStatusCode(fasthttp.StatusForbidden)
	} else {
		next(ctx)
	}
}
