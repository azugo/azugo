package azugo

import (
	"testing"

	"github.com/valyala/fasthttp"
)

func benchRequestCtx(method, uri string) *fasthttp.RequestCtx {
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(uri)

	return ctx
}

func BenchmarkContextAcquireRelease(b *testing.B) {
	app := NewTestApp()
	ctx := benchRequestCtx("GET", "/test")

	b.ReportAllocs()

	for b.Loop() {
		c := app.acquireCtx(app.defaultMux, "/test", ctx)
		app.releaseCtx(c)
	}
}

func BenchmarkContextQuery(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/user", func(ctx *Context) {
		_, _ = ctx.Query.Int("id")
		_, _ = ctx.Query.String("name")
	})

	ctx := benchRequestCtx("GET", "/user?id=15&name=John")

	b.ReportAllocs()

	for b.Loop() {
		m.Handler(ctx)
	}
}

func BenchmarkContextQueryValues(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/user", func(ctx *Context) {
		_ = ctx.Query.Values("tag")
	})

	ctx := benchRequestCtx("GET", "/user?tag=a,b,c&tag=d")

	b.ReportAllocs()

	for b.Loop() {
		m.Handler(ctx)
	}
}

func BenchmarkContextParams(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/user/{id}", func(ctx *Context) {
		_, _ = ctx.Params.Int("id")
	})

	ctx := benchRequestCtx("GET", "/user/15")

	b.ReportAllocs()

	for b.Loop() {
		m.Handler(ctx)
	}
}

func BenchmarkContextHeader(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/test", func(ctx *Context) {
		_ = ctx.Header.Get("X-Custom-Header")
		ctx.Header.Set("X-Response-Header", "value")
	})

	ctx := benchRequestCtx("GET", "/test")
	ctx.Request.Header.Set("X-Custom-Header", "value")

	b.ReportAllocs()

	for b.Loop() {
		m.Handler(ctx)
	}
}

func BenchmarkContextIP(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/test", func(ctx *Context) {
		_ = ctx.IP()
		_ = ctx.IsTrustedProxy()
	})

	ctx := benchRequestCtx("GET", "/test")

	b.ReportAllocs()

	for b.Loop() {
		m.Handler(ctx)
	}
}

func BenchmarkContextAccepts(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/test", func(ctx *Context) {
		_ = ctx.Accepts("application/json")
	})

	ctx := benchRequestCtx("GET", "/test")
	ctx.Request.Header.Set("Accept", "text/html, application/xhtml+xml, application/json;q=0.9, */*;q=0.8")

	b.ReportAllocs()

	for b.Loop() {
		m.Handler(ctx)
	}
}

func BenchmarkContextResponseJSON(b *testing.B) {
	type user struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
	}

	m := newMux(NewTestApp().App)
	m.Get("/json", func(ctx *Context) {
		ctx.JSON(user{ID: 1, Name: "John"})
	})

	ctx := benchRequestCtx("GET", "/json")

	b.ReportAllocs()

	for b.Loop() {
		m.Handler(ctx)
	}
}

func BenchmarkContextBodyJSON(b *testing.B) {
	type user struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	m := newMux(NewTestApp().App)
	m.Post("/json", func(ctx *Context) {
		var u user

		_ = ctx.Body.JSON(&u)
	})

	ctx := benchRequestCtx("POST", "/json")
	ctx.Request.Header.SetContentType("application/json")
	ctx.Request.SetBody([]byte(`{"name":"John","age":30}`))

	b.ReportAllocs()

	for b.Loop() {
		m.Handler(ctx)
	}
}
