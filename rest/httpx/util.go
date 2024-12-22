package httpx

import (
	"errors"
	"github.com/valyala/fasthttp"
)

const xForwardedFor = "X-Forwarded-For"

// GetFormValues returns the form values.
func GetFormValues(r *fasthttp.Request) (res map[string]any, err error) {
	res = map[string]any{}
	f := func(key, value []byte) {
		if len(value) == 0 {
			return
		}
		sk := string(key)
		sv := string(value)
		if v := res[sk]; v != nil {
			arr := v.([]string)
			res[sk] = append(arr, sv)
		} else {
			res[sk] = []string{sv}
		}
	}
	r.PostArgs().VisitAll(f)

	var formMap map[string][]string
	if form, err := r.MultipartForm(); err != nil {
		if !errors.Is(err, fasthttp.ErrNoMultipartForm) {
			return nil, err
		}
	} else {
		formMap = form.Value
	}

	for name, values := range formMap {
		if len(values) > 0 {
			if v := res[name]; v != nil {
				arr := v.([]string)
				arr = append(arr, values...)
			} else {
				tmp := make([]string, len(values))
				copy(tmp, values)
				res[name] = tmp
			}
		}
	}

	r.URI().QueryArgs().VisitAll(f)

	return res, nil
}

// GetRemoteAddr returns the peer address, supports X-Forward-For.
func GetRemoteAddr(ctx *fasthttp.RequestCtx) string {
	v := ctx.Request.Header.Peek(xForwardedFor)
	if len(v) > 0 {
		return string(v)
	}

	return ctx.RemoteAddr().String()
}
