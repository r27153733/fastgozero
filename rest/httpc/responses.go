package httpc

import (
	"bytes"
	"github.com/r27153733/fastgozero/core/mapping"
	"github.com/r27153733/fastgozero/fastext/bytesconv"
	"github.com/r27153733/fastgozero/rest/internal/encoding"
	"github.com/r27153733/fastgozero/rest/internal/header"
	"github.com/valyala/fasthttp"
	"io"
)

// Parse parses the response.
func Parse(resp *fasthttp.Response, val any) error {
	if err := ParseHeaders(resp, val); err != nil {
		return err
	}

	return ParseJsonBody(resp, val)
}

// ParseHeaders parses the response headers.
func ParseHeaders(resp *fasthttp.Response, val any) error {
	return encoding.ParseRespHeaders(&resp.Header, val)
}

// ParseJsonBody parses the response body, which should be in json content type.
func ParseJsonBody(resp *fasthttp.Response, val any) error {
	if isContentTypeJson(resp) {
		if resp.Header.ContentLength() > 0 {
			return mapping.UnmarshalJsonReader(bytes.NewReader(resp.Body()), val)
		}

		var buf bytes.Buffer
		if _, err := io.Copy(&buf, bytes.NewReader(resp.Body())); err != nil {
			return err
		}

		if buf.Len() > 0 {
			return mapping.UnmarshalJsonReader(&buf, val)
		}
	}

	return mapping.UnmarshalJsonMap(nil, val)
}

func isContentTypeJson(r *fasthttp.Response) bool {
	return bytes.Contains(r.Header.Peek(header.ContentType), bytesconv.SToB(header.ApplicationJson))
}
