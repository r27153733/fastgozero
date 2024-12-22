package pathvar

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestVars(t *testing.T) {
	expect := MapParams(map[string]string{
		"a": "1",
		"b": "2",
	})
	r := fasthttp.RequestCtx{}
	free := SetVars(&r, expect)
	vars := Vars(&r)
	vars.VisitAll(func(key, value string) {
		v, _ := expect[key]
		assert.Equal(t, v, value)
	})
	for k, v := range expect {
		get, ok := vars.Get(k)
		assert.True(t, ok)
		assert.Equal(t, v, get)
	}
	free()
	assert.Nil(t, Vars(&r))
}

func TestVarsNil(t *testing.T) {
	r := fasthttp.RequestCtx{}
	assert.Nil(t, Vars(&r))
}
