package httpx

import (
	"errors"
	"github.com/valyala/fasthttp"
	"mime/multipart"
)

const xForwardedFor = "X-Forwarded-For"

// GetFormValues returns the form values.
func GetFormValues(r *fasthttp.Request) (res map[string]any, err error) {
	var form *multipart.Form
	if form, err = r.MultipartForm(); err != nil {
		if !errors.Is(err, fasthttp.ErrNoMultipartForm) {
			return nil, err
		} else {
			return make(map[string]any), nil
		}
	}

	params := make(map[string]any, len(form.Value))
	for name, values := range form.Value {
		filtered := make([]string, 0, len(values))
		for _, v := range values {
			if len(v) > 0 {
				filtered = append(filtered, v)
			}
		}

		if len(filtered) > 0 {
			params[name] = filtered
		}
	}

	return params, nil
}

// GetRemoteAddr returns the peer address, supports X-Forward-For.
func GetRemoteAddr(ctx *fasthttp.RequestCtx) string {
	v := ctx.Request.Header.Peek(xForwardedFor)
	if len(v) > 0 {
		return string(v)
	}

	return ctx.RemoteAddr().String()
}
