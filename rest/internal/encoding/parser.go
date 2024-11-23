package encoding

import (
	"github.com/valyala/fasthttp"
	"github.com/zeromicro/go-zero/fastext"
	"net/textproto"

	"github.com/zeromicro/go-zero/core/mapping"
)

const headerKey = "header"

var headerUnmarshaler = mapping.NewUnmarshaler(headerKey, mapping.WithStringValues(),
	mapping.WithCanonicalKeyFunc(textproto.CanonicalMIMEHeaderKey))

// ParseHeaders parses the headers request.
func ParseHeaders(header *fasthttp.RequestHeader, v any) error {
	m := map[string]any{}

	header.VisitAll(func(k []byte, v []byte) {
		sk := fastext.B2s(k)
		if vv, ok := m[sk]; ok {
			switch tv := vv.(type) {
			case string:
				m[sk] = []string{tv, fastext.B2s(v)}
			case []string:
				m[sk] = append(tv, fastext.B2s(v))
			}
		} else {
			m[sk] = fastext.B2s(v)
		}
	})
	return headerUnmarshaler.Unmarshal(m, v)
}
