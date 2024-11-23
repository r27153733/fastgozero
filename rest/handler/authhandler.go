package handler

import (
	"errors"

	"github.com/golang-jwt/jwt/v4"
	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/fastext"
	"github.com/zeromicro/go-zero/rest/token"
)

const (
	jwtAudience    = "aud"
	jwtExpire      = "exp"
	jwtId          = "jti"
	jwtIssueAt     = "iat"
	jwtIssuer      = "iss"
	jwtNotBefore   = "nbf"
	jwtSubject     = "sub"
	noDetailReason = "no detail reason"
)

var (
	errInvalidToken = errors.New("invalid auth token")
	errNoClaims     = errors.New("no auth params")
)

type (
	// An AuthorizeOptions is authorize options.
	AuthorizeOptions struct {
		PrevSecret string
		Callback   UnauthorizedCallback
	}

	// UnauthorizedCallback defines the method of unauthorized callback.
	UnauthorizedCallback func(ctx *fasthttp.RequestCtx, err error)
	// AuthorizeOption defines the method to customize an AuthorizeOptions.
	AuthorizeOption func(opts *AuthorizeOptions)
)

// Authorize returns an authorization middleware.
func Authorize(secret string, opts ...AuthorizeOption) func(fasthttp.RequestHandler) fasthttp.RequestHandler {
	var authOpts AuthorizeOptions
	for _, opt := range opts {
		opt(&authOpts)
	}

	parser := token.NewTokenParser()
	return func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			tok, err := parser.ParseToken(&ctx.Request, secret, authOpts.PrevSecret)
			if err != nil {
				unauthorized(ctx, err, authOpts.Callback)
				return
			}

			if !tok.Valid {
				unauthorized(ctx, errInvalidToken, authOpts.Callback)
				return
			}

			claims, ok := tok.Claims.(jwt.MapClaims)
			if !ok {
				unauthorized(ctx, errNoClaims, authOpts.Callback)
				return
			}

			for k, v := range claims {
				switch k {
				case jwtAudience, jwtExpire, jwtId, jwtIssueAt, jwtIssuer, jwtNotBefore, jwtSubject:
					// ignore the standard claims
				default:
					//ctx.SetUserValue(k, v)
					free := fastext.SetUserValueCtx(ctx, k, v)
					defer free()
				}
			}

			next(ctx)
		}
	}
}

// WithPrevSecret returns an AuthorizeOption with setting previous secret.
func WithPrevSecret(secret string) AuthorizeOption {
	return func(opts *AuthorizeOptions) {
		opts.PrevSecret = secret
	}
}

// WithUnauthorizedCallback returns an AuthorizeOption with setting unauthorized callback.
func WithUnauthorizedCallback(callback UnauthorizedCallback) AuthorizeOption {
	return func(opts *AuthorizeOptions) {
		opts.Callback = callback
	}
}

func detailAuthLog(r *fasthttp.Request, reason string) {
	// discard dump error, only for debug purpose
	logx.Errorf("authorize failed: %s\n=> %+v", reason, r.String())
}

func unauthorized(ctx *fasthttp.RequestCtx, err error, callback UnauthorizedCallback) {
	if err != nil {
		detailAuthLog(&ctx.Request, err.Error())
	} else {
		detailAuthLog(&ctx.Request, noDetailReason)
	}

	// let callback go first, to make sure we respond with user-defined HTTP header
	if callback != nil {
		callback(ctx, err)
	}

	// if user not setting HTTP header, we set header with 401
	ctx.Response.SetStatusCode(fasthttp.StatusUnauthorized)
}
