package httpx

import (
	"bytes"
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"io"
	"reflect"
	"strings"
	"sync/atomic"

	"github.com/r27153733/fastgozero/core/mapping"
	"github.com/r27153733/fastgozero/core/validation"
	"github.com/r27153733/fastgozero/rest/internal/encoding"
	"github.com/r27153733/fastgozero/rest/internal/header"
	"github.com/r27153733/fastgozero/rest/router/pathvar"
	"github.com/valyala/fasthttp"
)

const (
	formKey           = "form"
	pathKey           = "path"
	maxMemory         = 32 << 20 // 32MB
	maxBodyLen        = 8 << 20  // 8MB
	separator         = ";"
	tokensInAttribute = 2
)

var (
	formUnmarshaler = mapping.NewUnmarshaler(
		formKey,
		mapping.WithStringValues(),
		mapping.WithOpaqueKeys(),
		mapping.WithFromArray())
	pathUnmarshaler = mapping.NewUnmarshaler(
		pathKey,
		mapping.WithStringValues(),
		mapping.WithOpaqueKeys())
	validator atomic.Value
)

// Validator defines the interface for validating the request.
type Validator interface {
	// Validate validates the request and parsed data.
	Validate(r *fasthttp.Request, data any) error
}

// Parse parses the request.
func Parse(r *fasthttp.RequestCtx, v any) error {
	kind := mapping.Deref(reflect.TypeOf(v)).Kind()
	if kind != reflect.Array && kind != reflect.Slice {
		if err := ParsePath(r, v); err != nil {
			return err
		}

		if err := ParseForm(&r.Request, v); err != nil {
			return err
		}

		if err := ParseHeaders(&r.Request, v); err != nil {
			return err
		}
	}

	if err := ParseJsonBody(&r.Request, v); err != nil {
		return err
	}

	if valid, ok := v.(validation.Validator); ok {
		return valid.Validate()
	} else if val := validator.Load(); val != nil {
		return val.(Validator).Validate(&r.Request, v)
	}

	return nil
}

// ParseHeaders parses the headers request.
func ParseHeaders(r *fasthttp.Request, v any) error {
	return encoding.ParseHeaders(&r.Header, v)
}

// ParseForm parses the form request.
func ParseForm(r *fasthttp.Request, v any) error {
	params, err := GetFormValues(r)
	if err != nil {
		return err
	}

	return formUnmarshaler.Unmarshal(params, v)
}

// ParseHeader parses the request header and returns a map.
func ParseHeader(headerValue string) map[string]string {
	ret := make(map[string]string)
	fields := strings.Split(headerValue, separator)

	for _, field := range fields {
		field = strings.TrimSpace(field)
		if len(field) == 0 {
			continue
		}

		kv := strings.SplitN(field, "=", tokensInAttribute)
		if len(kv) != tokensInAttribute {
			continue
		}

		ret[kv[0]] = kv[1]
	}

	return ret
}

// ParseJsonBody parses the post request which contains json in body.
func ParseJsonBody(r *fasthttp.Request, v any) error {
	if withJsonBody(r) {
		var reader io.Reader
		if !r.IsBodyStream() {
			reader = bytes.NewReader(r.Body())
		} else {
			reader = io.LimitReader(r.BodyStream(), maxBodyLen)
		}

		return mapping.UnmarshalJsonReader(reader, v)
	}

	return mapping.UnmarshalJsonMap(nil, v)
}

// ParsePath parses the symbols reside in url path.
// Like http://localhost/bag/:name
func ParsePath(r *fasthttp.RequestCtx, v any) error {
	vars := pathvar.Vars(r)
	if vars == nil {
		return nil
	}
	return pathUnmarshaler.UnmarshalValuer(Str2StrValue(vars.Get), v)
}

type Str2StrValue func(string) (string, bool)

func (f Str2StrValue) Value(key string) (any, bool) {
	v, ok := f(key)
	return v, ok
}

// SetValidator sets the validator.
// The validator is used to validate the request, only called in Parse,
// not in ParseHeaders, ParseForm, ParseHeader, ParseJsonBody, ParsePath.
func SetValidator(val Validator) {
	validator.Store(val)
}

func withJsonBody(r *fasthttp.Request) bool {
	return r.Header.ContentLength() > 0 && strings.Contains(bytesconv.BToS(r.Header.Peek(header.ContentType)), header.ApplicationJson)
}
