package pathvar

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestVars(t *testing.T) {
	expect := map[string]string{
		"a": "1",
		"b": "2",
	}
	r := fasthttp.RequestCtx{}
	free := SetVars(&r, expect)
	assert.EqualValues(t, expect, Vars(&r))
	free()
	assert.Nil(t, Vars(&r))
}

func TestVarsNil(t *testing.T) {
	r := fasthttp.RequestCtx{}
	assert.Nil(t, Vars(&r))
}

func TestContextKey(t *testing.T) {
	ck := contextKey("hello")
	assert.True(t, strings.Contains(ck.String(), "hello"))
}
