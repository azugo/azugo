package azugo

import (
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestMuxGroup(t *testing.T) {
	handlerFunc := func(*Context) {}

	var (
		muxUseCalled   int
		groupUseCalled int
	)

	m := newMux(NewTestApp().App)
	m.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			muxUseCalled++
			next(ctx)
		}
	})
	g := m.Group("/group")
	{
		g.Use(func(next RequestHandler) RequestHandler {
			return func(ctx *Context) {
				groupUseCalled++
				next(ctx)
			}
		})
		g.Get("/path", handlerFunc)
	}

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/group/path")

	m.Handler(ctx)
	qt.Check(t, qt.Equals(muxUseCalled, 1))
	qt.Check(t, qt.Equals(groupUseCalled, 1))
}

func BenchmarkAllowed(b *testing.B) {
	handlerFunc := func(*Context) {}

	m := newMux(NewTestApp().App)
	m.Post("/path", handlerFunc)
	m.Get("/path", handlerFunc)

	b.Run("Global", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = m.Allowed("*", fasthttp.MethodOptions)
		}
	})
	b.Run("Path", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = m.Allowed("/path", fasthttp.MethodOptions)
		}
	})
}

func BenchmarkRouterGet(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/hello", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/hello")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.Handler(ctx)
	}
}

func BenchmarkRouterParams(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/{id}", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/hello")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.Handler(ctx)
	}
}

func BenchmarkRouterANY(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/data", func(ctx *Context) {})
	m.Any("/", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.Handler(ctx)
	}
}

func BenchmarkRouterGet_ANY(b *testing.B) {
	var (
		resp    = "Bench GET"
		respANY = "Bench GET (ANY)"
	)

	m := newMux(NewTestApp().App)
	m.Get("/", func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text(resp)
	})
	m.Any("/", func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text(respANY)
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("UNICORN")
	ctx.Request.SetRequestURI("/")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.Handler(ctx)
	}
}

func BenchmarkRouterNotFound(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/bench", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/notfound")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.Handler(ctx)
	}
}

func BenchmarkRouterFindCaseInsensitive(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/bench", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/BenCh/.")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.Handler(ctx)
	}
}

func BenchmarkRouterRedirectTrailingSlash(b *testing.B) {
	m := newMux(NewTestApp().App)
	m.Get("/bench/", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/bench")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.Handler(ctx)
	}
}

func Benchmark_Get(b *testing.B) {
	handler := func(ctx *Context) {}

	m := newMux(NewTestApp().App)
	m.Get("/", handler)
	m.Get("/plaintext", handler)
	m.Get("/json", handler)
	m.Get("/fortune", handler)
	m.Get("/fortune-quick", handler)
	m.Get("/db", handler)
	m.Get("/queries", handler)
	m.Get("/update", handler)

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/update")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		m.Handler(ctx)
	}
}
