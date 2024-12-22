package encoding

import (
	"net/textproto"

	"github.com/r27153733/fastgozero/core/mapping"
	"github.com/valyala/fasthttp"
)

const headerKey = "header"

var headerUnmarshaler = mapping.NewUnmarshaler(headerKey, mapping.WithStringValues(),
	mapping.WithCanonicalKeyFunc(textproto.CanonicalMIMEHeaderKey))

// ParseHeaders parses the headers request.
func ParseHeaders(header *fasthttp.RequestHeader, v any) error {
	m := map[string]any{}

	header.VisitAll(func(k []byte, v []byte) {
		sk := string(k)
		if vv, ok := m[sk]; ok {
			switch tv := vv.(type) {
			case string:
				m[sk] = []string{tv, string(v)}
			case []string:
				m[sk] = append(tv, string(v))
			}
		} else {
			m[sk] = string(v)
		}
	})
	return headerUnmarshaler.Unmarshal(m, v)
}
