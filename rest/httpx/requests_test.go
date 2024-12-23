package httpx

import (
	"errors"
	"github.com/valyala/fasthttp"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/r27153733/fastgozero/rest/internal/header"
	"github.com/r27153733/fastgozero/rest/router/pathvar"
	"github.com/stretchr/testify/assert"
)

func TestParseForm(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		var v struct {
			Name    string  `form:"name"`
			Age     int     `form:"age"`
			Percent float64 `form:"percent,optional"`
		}
		r := new(fasthttp.RequestCtx)
		r.Request.Header.SetMethod(fasthttp.MethodGet)
		r.Request.SetRequestURI("/a?name=hello&age=18&percent=3.4")

		assert.Nil(t, Parse(r, &v))
		assert.Equal(t, "hello", v.Name)
		assert.Equal(t, 18, v.Age)
		assert.Equal(t, 3.4, v.Percent)
	})

	t.Run("no value", func(t *testing.T) {
		var v struct {
			NoValue string `form:"noValue,optional"`
		}
		r := new(fasthttp.RequestCtx)
		r.Request.Header.SetMethod(fasthttp.MethodGet)
		r.Request.SetRequestURI("/a?name=hello&age=18&percent=3.4&statuses=try&statuses=done&singleValue=one")

		assert.Nil(t, Parse(r, &v))
		assert.Equal(t, 0, len(v.NoValue))
	})
}

func TestParseFormArray(t *testing.T) {
	t.Run("slice", func(t *testing.T) {
		var v struct {
			Name    []string  `form:"name"`
			Age     []int     `form:"age"`
			Percent []float64 `form:"percent,optional"`
		}

		r := new(fasthttp.RequestCtx)
		r.Request.Header.SetMethod(fasthttp.MethodGet)
		r.Request.SetRequestURI("/a?name=hello&name=world&age=18&age=19&percent=3.4&percent=4.5")

		if assert.NoError(t, Parse(r, &v)) {
			assert.ElementsMatch(t, []string{"hello", "world"}, v.Name)
			assert.ElementsMatch(t, []int{18, 19}, v.Age)
			assert.ElementsMatch(t, []float64{3.4, 4.5}, v.Percent)
		}
	})

	t.Run("slice with single value", func(t *testing.T) {
		var v struct {
			Name    []string  `form:"name"`
			Age     []int     `form:"age"`
			Percent []float64 `form:"percent,optional"`
		}

		r := new(fasthttp.RequestCtx)
		r.Request.Header.SetMethod(fasthttp.MethodGet)
		r.Request.SetRequestURI("/a?name=hello&age=18&percent=3.4")

		if assert.NoError(t, Parse(r, &v)) {
			assert.ElementsMatch(t, []string{"hello"}, v.Name)
			assert.ElementsMatch(t, []int{18}, v.Age)
			assert.ElementsMatch(t, []float64{3.4}, v.Percent)
		}
	})

	t.Run("slice with empty and non-empty", func(t *testing.T) {
		var v struct {
			Name []string `form:"name"`
		}

		r := new(fasthttp.RequestCtx)
		r.Request.Header.SetMethod(fasthttp.MethodGet)
		r.Request.SetRequestURI("/a?name=&name=1")

		if assert.NoError(t, Parse(r, &v)) {
			assert.ElementsMatch(t, []string{"1"}, v.Name)
		}
	})
}

func TestParseForm_Error(t *testing.T) {
	var v struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}

	r := new(fasthttp.RequestCtx)
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	r.Request.SetRequestURI("/a?name=hello;")

	assert.NotNil(t, ParseForm(&r.Request, &v))
}

func TestParseHeader(t *testing.T) {
	tests := []struct {
		name   string
		value  string
		expect map[string]string
	}{
		{
			name:   "empty",
			value:  "",
			expect: map[string]string{},
		},
		{
			name:   "regular",
			value:  "key=value",
			expect: map[string]string{"key": "value"},
		},
		{
			name:   "next empty",
			value:  "key=value;",
			expect: map[string]string{"key": "value"},
		},
		{
			name:   "regular",
			value:  "key=value;foo",
			expect: map[string]string{"key": "value"},
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			m := ParseHeader(test.value)
			assert.EqualValues(t, test.expect, m)
		})
	}
}

func TestParsePath(t *testing.T) {
	var v struct {
		Name string `path:"name"`
		Age  int    `path:"age"`
	}

	r := new(fasthttp.RequestCtx)
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	r.Request.SetRequestURI("/")
	pathvar.SetVars(r, pathvar.MapParams(map[string]string{
		"name": "foo",
		"age":  "18",
	}))
	err := Parse(r, &v)
	assert.Nil(t, err)
	assert.Equal(t, "foo", v.Name)
	assert.Equal(t, 18, v.Age)
}

func TestParsePath_Error(t *testing.T) {
	var v struct {
		Name string `path:"name"`
		Age  int    `path:"age"`
	}

	r := new(fasthttp.RequestCtx)
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	r.Request.SetRequestURI("/")
	pathvar.SetVars(r, pathvar.MapParams(map[string]string{
		"name": "foo",
	}))
	assert.NotNil(t, Parse(r, &v))
}

func TestParseFormOutOfRange(t *testing.T) {
	var v struct {
		Age int `form:"age,range=[10:20)"`
	}

	tests := []struct {
		url  string
		pass bool
	}{
		{
			url:  "/a?age=5",
			pass: false,
		},
		{
			url:  "/a?age=10",
			pass: true,
		},
		{
			url:  "/a?age=15",
			pass: true,
		},
		{
			url:  "/a?age=20",
			pass: false,
		},
		{
			url:  "/a?age=28",
			pass: false,
		},
	}

	for _, test := range tests {
		r := new(fasthttp.RequestCtx)
		r.Request.SetRequestURI(test.url)
		r.Request.Header.SetMethod(fasthttp.MethodGet)

		err := Parse(r, &v)
		if test.pass {
			assert.Nil(t, err)
		} else {
			assert.NotNil(t, err)
		}
	}
}

func TestParseMultipartForm(t *testing.T) {
	var v struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}

	body := strings.Replace(`----------------------------220477612388154780019383
Content-Disposition: form-data; name="name"

kevin
----------------------------220477612388154780019383
Content-Disposition: form-data; name="age"

18
----------------------------220477612388154780019383--`, "\n", "\r\n", -1)

	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/")
	r.Request.Header.SetMethod(fasthttp.MethodPost)
	r.Request.Header.Set(ContentType, "multipart/form-data; boundary=--------------------------220477612388154780019383")
	r.Request.SetBodyString(body)
	r.Request.Header.SetContentLength(len(r.Request.Body()))

	assert.Nil(t, Parse(r, &v))
	assert.Equal(t, "kevin", v.Name)
	assert.Equal(t, 18, v.Age)
}

func TestParseMultipartFormWrongBoundary(t *testing.T) {
	var v struct {
		Name string `form:"name"`
		Age  int    `form:"age"`
	}

	body := strings.Replace(`----------------------------22047761238815478001938
Content-Disposition: form-data; name="name"

kevin
----------------------------22047761238815478001938
Content-Disposition: form-data; name="age"

18
----------------------------22047761238815478001938--`, "\n", "\r\n", -1)

	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/")
	r.Request.Header.SetMethod(fasthttp.MethodPost)
	r.Request.Header.Set(ContentType, "multipart/form-data; boundary=--------------------------220477612388154780019383")
	r.Request.SetBodyString(body)
	r.Request.Header.SetContentLength(len(r.Request.Body()))

	assert.NotNil(t, Parse(r, &v))
}

func TestParseJsonBody(t *testing.T) {
	t.Run("has body", func(t *testing.T) {
		var v struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		body := `{"name":"kevin", "age": 18}`

		r := new(fasthttp.RequestCtx)
		r.Request.SetRequestURI("/")
		r.Request.Header.SetMethod(fasthttp.MethodPost)
		r.Request.Header.Set(ContentType, header.JsonContentType)
		r.Request.SetBodyString(body)
		r.Request.Header.SetContentLength(len(r.Request.Body()))

		if assert.NoError(t, Parse(r, &v)) {
			assert.Equal(t, "kevin", v.Name)
			assert.Equal(t, 18, v.Age)
		}
	})

	t.Run("bad body", func(t *testing.T) {
		var v struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		body := `{"name":"kevin", "ag": 18}`

		// Create a new fasthttp.RequestCtx
		r := new(fasthttp.RequestCtx)
		r.Request.SetRequestURI("/")
		r.Request.Header.SetMethod(fasthttp.MethodPost)
		r.Request.Header.Set(ContentType, header.JsonContentType)
		r.Request.SetBodyString(body)

		assert.Error(t, Parse(r, &v))
	})

	t.Run("hasn't body", func(t *testing.T) {
		var v struct {
			Name string `json:"name,optional"`
			Age  int    `json:"age,optional"`
		}

		// Create a new fasthttp.RequestCtx with no body
		r := new(fasthttp.RequestCtx)
		r.Request.SetRequestURI("/")
		r.Request.Header.SetMethod(fasthttp.MethodGet)
		r.Request.SetBody([]byte{})

		assert.Nil(t, Parse(r, &v))
		assert.Equal(t, "", v.Name)
		assert.Equal(t, 0, v.Age)
	})

	t.Run("array body", func(t *testing.T) {
		var v []struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		}

		body := `[{"name":"kevin", "age": 18}]`

		r := new(fasthttp.RequestCtx)
		r.Request.SetRequestURI("/")
		r.Request.Header.SetMethod(fasthttp.MethodPost)
		r.Request.Header.Set(ContentType, header.JsonContentType)
		r.Request.SetBodyString(body)
		r.Request.Header.SetContentLength(len(r.Request.Body()))

		assert.NoError(t, Parse(r, &v))
		assert.Equal(t, 1, len(v))
		assert.Equal(t, "kevin", v[0].Name)
		assert.Equal(t, 18, v[0].Age)
	})

	t.Run("form and array body", func(t *testing.T) {
		var v []struct {
			// we can only ignore the form tag,
			// because if the value is a slice, it should be in the body,
			// but it's hard to detect when we treat it as a json body.
			Product string `form:"product"`
			Name    string `json:"name"`
			Age     int    `json:"age"`
		}

		body := `[{"name":"apple", "age": 18}]`

		// Create a new fasthttp.RequestCtx
		r := new(fasthttp.RequestCtx)
		r.Request.SetRequestURI("/a?product=tree")
		r.Request.Header.SetMethod(fasthttp.MethodPost)
		r.Request.Header.Set(ContentType, header.JsonContentType)
		r.Request.SetBodyString(body)
		r.Request.Header.SetContentLength(len(r.Request.Body()))

		assert.NoError(t, Parse(r, &v))
		assert.Equal(t, 1, len(v))
		assert.Equal(t, "apple", v[0].Name)
		assert.Equal(t, 18, v[0].Age)
	})
}

func TestParseRequired(t *testing.T) {
	v := struct {
		Name    string  `form:"name"`
		Percent float64 `form:"percent"`
	}{}

	// Create a new fasthttp.RequestCtx
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/a?name=hello")
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	r.Request.SetBody([]byte{})

	// Parse the request and assert the error is not nil
	assert.NotNil(t, Parse(r, &v)) // Expecting an error due to missing 'percent'
}

func TestParseOptions(t *testing.T) {
	v := struct {
		Position int8 `form:"pos,options=1|2"`
	}{}

	// Create a new fasthttp.RequestCtx
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/a?pos=4")
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	r.Request.SetBody([]byte{})

	// Parse the request and assert the error is not nil
	assert.NotNil(t, Parse(r, &v)) // Expecting an error because the value 4 is not in the options [1|2]
}

func TestParseHeaders(t *testing.T) {
	type AnonymousStruct struct {
		XRealIP string `header:"x-real-ip"`
		Accept  string `header:"Accept,optional"`
	}
	v := struct {
		Name          string   `header:"name,optional"`
		Percent       string   `header:"percent"`
		Addrs         []string `header:"addrs"`
		XForwardedFor string   `header:"X-Forwarded-For,optional"`
		AnonymousStruct
	}{}
	request := new(fasthttp.RequestCtx)
	request.Request.Header.SetMethod(fasthttp.MethodPost)
	request.Request.Header.Set("name", "chenquan")
	request.Request.Header.Set("percent", "1")
	request.Request.Header.Add("addrs", "addr1")
	request.Request.Header.Add("addrs", "addr2")
	request.Request.Header.Add("X-Forwarded-For", "10.0.10.11")
	request.Request.Header.Add("x-real-ip", "10.0.11.10")
	request.Request.Header.Add("Accept", header.JsonContentType)

	err := ParseHeaders(&request.Request, &v)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "chenquan", v.Name)
	assert.Equal(t, "1", v.Percent)
	assert.Equal(t, []string{"addr1", "addr2"}, v.Addrs)
	assert.Equal(t, "10.0.10.11", v.XForwardedFor)
	assert.Equal(t, "10.0.11.10", v.XRealIP)
	assert.Equal(t, header.JsonContentType, v.Accept)
}

func TestParseHeaders_Error(t *testing.T) {
	v := struct {
		Name string `header:"name"`
		Age  int    `header:"age"`
	}{}

	// Create a new fasthttp.RequestCtx
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/")
	r.Request.Header.SetMethod(fasthttp.MethodPost)

	// Set the headers in the request context
	r.Request.Header.Set("name", "foo")

	// Parse the request and assert the error is not nil
	// Since 'age' is missing, we expect an error
	assert.NotNil(t, Parse(r, &v))
}

func TestParseWithValidator(t *testing.T) {
	// Set a mock validator for testing, and defer to reset it.
	SetValidator(mockValidator{})
	defer SetValidator(mockValidator{nop: true})

	var v struct {
		Name    string  `form:"name"`
		Age     int     `form:"age"`
		Percent float64 `form:"percent,optional"`
	}

	// Create a new fasthttp.RequestCtx for testing
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/a?name=hello&age=18&percent=3.4")
	r.Request.Header.SetMethod(fasthttp.MethodGet)

	// Simulate the URL query parameters
	r.Request.Header.Set(ContentType, "application/x-www-form-urlencoded") // for form data
	r.URI().SetQueryString("name=hello&age=18&percent=3.4")

	// Call the Parse function, passing the fasthttp.RequestCtx
	if assert.NoError(t, Parse(r, &v)) {
		// Validate the parsed values
		assert.Equal(t, "hello", v.Name)
		assert.Equal(t, 18, v.Age)
		assert.Equal(t, 3.4, v.Percent)
	}
}

func TestParseWithValidatorWithError(t *testing.T) {
	// Set a mock validator for testing, and defer to reset it.
	SetValidator(mockValidator{})
	defer SetValidator(mockValidator{nop: true})

	var v struct {
		Name    string  `form:"name"`
		Age     int     `form:"age"`
		Percent float64 `form:"percent,optional"`
	}

	// Create a new fasthttp.RequestCtx for testing
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/a?name=world&age=18&percent=3.4")
	r.Request.Header.SetMethod(fasthttp.MethodGet)

	// Call the Parse function, passing the fasthttp.RequestCtx
	// Here, we simulate an error scenario (mockValidator should cause an error)
	assert.Error(t, Parse(r, &v))
}

func TestParseWithValidatorRequest(t *testing.T) {
	SetValidator(mockValidator{})
	defer SetValidator(mockValidator{nop: true})

	var v mockRequest
	// Create a new fasthttp.RequestCtx for testing
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/a?&age=18")
	r.Request.Header.SetMethod(fasthttp.MethodGet)

	// Simulate the URL query parameters
	r.URI().SetQueryString("age=18")

	// Call the Parse function with the fasthttp request context
	// Here, we expect the mock validator to cause an error.
	assert.Error(t, Parse(r, &v))
}

func TestParseFormWithDot(t *testing.T) {
	var v struct {
		Age int `form:"user.age"`
	}

	// Create a new fasthttp.RequestCtx for testing
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/a?user.age=18")
	r.Request.Header.SetMethod(fasthttp.MethodGet)

	// Simulate the URL query parameters
	r.URI().SetQueryString("user.age=18")

	// Call the Parse function with the fasthttp request context
	// Check if the parsing works as expected
	assert.NoError(t, Parse(r, &v))
	assert.Equal(t, 18, v.Age)
}

func TestParsePathWithDot(t *testing.T) {
	var v struct {
		Name string `path:"name.val"`
		Age  int    `path:"age.val"`
	}

	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/")
	r.Request.Header.SetMethod(fasthttp.MethodGet)
	pathvar.SetVars(r, pathvar.MapParams(map[string]string{
		"name.val": "foo",
		"age.val":  "18",
	}))
	err := Parse(r, &v)
	assert.Nil(t, err)
	assert.Equal(t, "foo", v.Name)
	assert.Equal(t, 18, v.Age)
}

func TestParseWithFloatPtr(t *testing.T) {
	t.Run("has float32 pointer", func(t *testing.T) {
		var v struct {
			WeightFloat32 *float32 `json:"weightFloat32,optional"`
		}
		body := `{"weightFloat32": 3.2}`
		r := new(fasthttp.RequestCtx)
		r.Request.SetRequestURI("/")
		r.Request.Header.SetMethod(fasthttp.MethodPost)
		r.Request.SetBodyString(body)
		r.Request.Header.SetContentLength(len(body))
		r.Request.Header.Set(ContentType, header.JsonContentType)

		if assert.NoError(t, Parse(r, &v)) {
			assert.Equal(t, float32(3.2), *v.WeightFloat32)
		}
	})
}

func TestParseWithEscapedParams(t *testing.T) {
	t.Run("escaped", func(t *testing.T) {
		var v struct {
			Dev string `form:"dev"`
		}
		r := new(fasthttp.RequestCtx)
		r.Request.SetRequestURI("http://127.0.0.1/api/v2/dev/test?dev=se205%5fy1205%5fj109%26verRelease=v01%26iid1=863494061186673%26iid2=863494061186681%26mcc=636%26mnc=1")
		r.Request.Header.SetMethod(fasthttp.MethodGet)

		if assert.NoError(t, Parse(r, &v)) {
			assert.Equal(t, "se205_y1205_j109&verRelease=v01&iid1=863494061186673&iid2=863494061186681&mcc=636&mnc=1", v.Dev)
		}
	})
}

func TestCustomUnmarshalerStructRequest(t *testing.T) {
	reqBody := `{"name": "hello"}`
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("/")
	r.Request.Header.SetMethod(fasthttp.MethodPost)
	r.Request.SetBodyString(reqBody)
	r.Request.Header.SetContentLength(len(reqBody))
	r.Request.Header.Set(ContentType, header.JsonContentType)
	v := struct {
		Foo *mockUnmarshaler `json:"name"`
	}{}
	assert.Nil(t, Parse(r, &v))
	assert.Equal(t, "hello", v.Foo.Name)
}

func BenchmarkParseRaw(b *testing.B) {
	r, err := http.NewRequest(http.MethodGet, "http://hello.com/a?name=hello&age=18&percent=3.4", http.NoBody)
	if err != nil {
		b.Fatal(err)
	}

	for i := 0; i < b.N; i++ {
		v := struct {
			Name    string  `form:"name"`
			Age     int     `form:"age"`
			Percent float64 `form:"percent,optional"`
		}{}

		v.Name = r.FormValue("name")
		v.Age, err = strconv.Atoi(r.FormValue("age"))
		if err != nil {
			b.Fatal(err)
		}
		v.Percent, err = strconv.ParseFloat(r.FormValue("percent"), 64)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkParseAuto(b *testing.B) {
	r := new(fasthttp.RequestCtx)
	r.Request.SetRequestURI("http://hello.com/a?name=hello&age=18&percent=3.4")
	r.Request.Header.SetMethod(fasthttp.MethodGet)

	for i := 0; i < b.N; i++ {
		v := struct {
			Name    string  `form:"name"`
			Age     int     `form:"age"`
			Percent float64 `form:"percent,optional"`
		}{}
		if err := Parse(r, &v); err != nil {
			b.Fatal(err)
		}
	}
}

type mockValidator struct {
	nop bool
}

func (m mockValidator) Validate(r *fasthttp.Request, data any) error {
	if m.nop {
		return nil
	}

	if string(r.URI().Path()) == "/a" {
		val := reflect.ValueOf(data).Elem().FieldByName("Name").String()
		if val != "hello" {
			return errors.New("name is not hello")
		}
	}

	return nil
}

type mockRequest struct {
	Name string `json:"name,optional"`
}

func (m mockRequest) Validate() error {
	if m.Name != "hello" {
		return errors.New("name is not hello")
	}

	return nil
}

type mockUnmarshaler struct {
	Name string
}

func (m *mockUnmarshaler) UnmarshalJSON(b []byte) error {
	m.Name = string(b)
	return nil
}
