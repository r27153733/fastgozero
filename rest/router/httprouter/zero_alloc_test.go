package httprouter

import (
	"github.com/valyala/fasthttp"
	"testing"
)

func TestZeroAlloc(t *testing.T) {
	//allocProfile := pprof.Lookup("allocs")
	//
	//f3, err := os.Create("alloc.pprof")
	//if err != nil {
	//	panic(err)
	//}
	//defer f3.Close()

	// Create the router
	router := New()
	_ = router.Handle(fasthttp.MethodGet, "/api/:user/:name", func(ctx *fasthttp.RequestCtx) {
		// Handler does nothing for this benchmark
	})

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/a/b")
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)

	f := testing.AllocsPerRun(100, func() {
		router.ServeHTTP(ctx)
	})

	//if err := allocProfile.WriteTo(f3, 0); err != nil {
	//	panic(err)
	//}

	if f != 0 {
		t.Fatal()
	}
}
