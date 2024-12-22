package internal

import (
	"errors"
	"testing"

	"github.com/r27153733/fastgozero/rest/router/pathvar"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestNewRequestParserNoVar(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)

	parser, err := NewRequestParser(req, nil)
	assert.Nil(t, err)
	assert.NotNil(t, parser)
}

func TestNewRequestParserWithVars(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)

	pathvar.SetVars(req, pathvar.MapParams(map[string]string{"a": "b"}))
	parser, err := NewRequestParser(req, nil)
	assert.Nil(t, err)
	assert.NotNil(t, parser)
}

func TestNewRequestParserNoVarWithBody(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)
	req.Request.SetBodyString(`{"a": "b"}`)
	req.Request.Header.SetContentLength(len(req.Request.Body()))

	parser, err := NewRequestParser(req, nil)
	assert.Nil(t, err)
	assert.NotNil(t, parser)
}

func TestNewRequestParserWithNegativeContentLength(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)
	req.Request.SetBodyString(`{"a": "b"}`)
	req.Request.Header.SetContentLength(-1)

	parser, err := NewRequestParser(req, nil)
	assert.Nil(t, err)
	assert.NotNil(t, parser)
}

func TestNewRequestParserWithVarsWithBody(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)
	req.Request.SetBodyString(`{"a": "b"}`)
	req.Request.Header.SetContentLength(len(req.Request.Body()))

	pathvar.SetVars(req, pathvar.MapParams(map[string]string{"c": "d"}))
	parser, err := NewRequestParser(req, nil)
	assert.Nil(t, err)
	assert.NotNil(t, parser)
}

func TestNewRequestParserWithVarsWithWrongBody(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)
	req.Request.SetBodyString(`{"a": "b"`)
	req.Request.Header.SetContentLength(len(req.Request.Body()))

	pathvar.SetVars(req, pathvar.MapParams(map[string]string{"c": "d"}))
	parser, err := NewRequestParser(req, nil)
	assert.NotNil(t, err)
	assert.Nil(t, parser)
}

func TestNewRequestParserWithForm(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)
	req.Request.SetRequestURI("/val?a=b")

	parser, err := NewRequestParser(req, nil)
	assert.Nil(t, err)
	assert.NotNil(t, parser)
}

func TestNewRequestParserWithNilBody(t *testing.T) {
	req := new(fasthttp.RequestCtx)
	req.Request.Header.SetMethod(fasthttp.MethodGet)
	req.Request.SetRequestURI("/val?a=b")

	parser, err := NewRequestParser(req, nil)
	assert.Nil(t, err)
	assert.NotNil(t, parser)
}

//func TestNewRequestParserWithBadBody(t *testing.T) {
//	req := new(fasthttp.RequestCtx)
//	req.Request.Header.SetMethod(fasthttp.MethodGet)
//	req.Request.SetRequestURI("/val?a=b")
//	req.Request.SetBodyStream(badBody{}, -1)
//
//	parser, err := NewRequestParser(req, nil)
//	assert.Nil(t, err)
//	assert.NotNil(t, parser)
//}

//func TestNewRequestParserWithBadForm(t *testing.T) {
//	req := new(fasthttp.RequestCtx)
//	req.Request.Header.SetMethod(fasthttp.MethodGet)
//	req.Request.SetRequestURI("/val?a%1=b")
//
//	parser, err := NewRequestParser(req, nil)
//	assert.NotNil(t, err)
//	assert.Nil(t, parser)
//}

func TestRequestParser_buildJsonRequestParser(t *testing.T) {
	parser, err := buildJsonRequestParser(map[string]any{"a": make(chan int)}, nil)
	assert.NotNil(t, err)
	assert.Nil(t, parser)
}

type badBody struct{}

func (badBody) Read([]byte) (int, error) { return 0, errors.New("something bad") }
func (badBody) Close() error             { return nil }
