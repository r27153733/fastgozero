package internal

import (
	"bytes"
	"encoding/json"
	"io"

	"github.com/fullstorydev/grpcurl"
	"github.com/golang/protobuf/jsonpb"
	"github.com/r27153733/fastgozero/rest/httpx"
	"github.com/r27153733/fastgozero/rest/router/pathvar"
	"github.com/valyala/fasthttp"
)

// NewRequestParser creates a new request parser from the given http.Request and resolver.
func NewRequestParser(r *fasthttp.RequestCtx, resolver jsonpb.AnyResolver) (grpcurl.RequestParser, error) {
	vars := pathvar.Vars(r)

	params, err := httpx.GetFormValues(&r.Request)
	if err != nil {
		return nil, err
	}

	vars.VisitAll(func(k, v string) {
		params[k] = v
	})

	body, ok := getBody(&r.Request)
	if !ok {
		return buildJsonRequestParser(params, resolver)
	}

	if len(params) == 0 {
		return grpcurl.NewJSONRequestParser(body, resolver), nil
	}

	m := make(map[string]any)
	if err := json.NewDecoder(body).Decode(&m); err != nil && err != io.EOF {
		return nil, err
	}

	for k, v := range params {
		m[k] = v
	}

	return buildJsonRequestParser(m, resolver)
}

func buildJsonRequestParser(m map[string]any, resolver jsonpb.AnyResolver) (
	grpcurl.RequestParser, error) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(m); err != nil {
		return nil, err
	}

	return grpcurl.NewJSONRequestParser(&buf, resolver), nil
}

func getBody(r *fasthttp.Request) (io.Reader, bool) {
	if r.Header.ContentLength() == 0 {
		return nil, false
	}

	if !r.IsBodyStream() {
		return bytes.NewReader(r.Body()), true
	} else {
		return r.BodyStream(), true
	}
}
