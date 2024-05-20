package azugo

import (
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestRouterGroupAPI(t *testing.T) {
	var handled, get, head, post, put, patch, delete, connect, options, trace, anyHandled bool

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	g := a.Group("/v1")

	g.Get("/GET", func(ctx *Context) {
		get = true
	})
	g.Head("/HEAD", func(ctx *Context) {
		head = true
	})
	g.Post("/POST", func(ctx *Context) {
		post = true
	})
	g.Put("/PUT", func(ctx *Context) {
		put = true
	})
	g.Patch("/PATCH", func(ctx *Context) {
		patch = true
	})
	g.Delete("/DELETE", func(ctx *Context) {
		delete = true
	})
	g.Connect("/CONNECT", func(ctx *Context) {
		connect = true
	})
	g.Options("/OPTIONS", func(ctx *Context) {
		options = true
	})
	g.Trace("/TRACE", func(ctx *Context) {
		trace = true
	})
	g.Any("/ANY", func(ctx *Context) {
		anyHandled = true
	})
	g.Handle(fasthttp.MethodGet, "/Handler", func(ctx *Context) {
		handled = true
	})

	resp, err := a.TestClient().Get("/v1/GET")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(get), qt.Commentf("GET route not handled"))

	resp, err = a.TestClient().Head("/v1/HEAD")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(head), qt.Commentf("HEAD route not handled"))

	resp, err = a.TestClient().Post("/v1/POST", nil)
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(post), qt.Commentf("POST route not handled"))

	resp, err = a.TestClient().Put("/v1/PUT", nil)
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(put), qt.Commentf("PUT route not handled"))

	resp, err = a.TestClient().Patch("/v1/PATCH", nil)
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(patch), qt.Commentf("PATCH route not handled"))

	resp, err = a.TestClient().Delete("/v1/DELETE")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(delete), qt.Commentf("DELETE route not handled"))

	resp, err = a.TestClient().Connect("/v1/CONNECT")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(connect), qt.Commentf("CONNECT route not handled"))

	resp, err = a.TestClient().Options("/v1/OPTIONS")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(options), qt.Commentf("OPTIONS route not handled"))

	resp, err = a.TestClient().Trace("/v1/TRACE")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(trace), qt.Commentf("TRACE route not handled"))

	resp, err = a.TestClient().Get("/v1/Handler")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(handled), qt.Commentf("Handler route not handled"))

	for _, method := range httpMethods {
		resp, err = a.TestClient().Call(method, "/v1/ANY", nil)
		fasthttp.ReleaseResponse(resp)
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.IsTrue(anyHandled), qt.Commentf("ANY route not handled"))
		anyHandled = false
	}
}

func TestRouterNestedGroups(t *testing.T) {
	var handled1, handled2, handled3 bool

	a := NewTestApp()

	g1 := a.Group("/foo")
	g2 := g1.Group("/bar")
	g3 := g1.Group("/baz")

	g1.Get("/", func(ctx *Context) {
		handled1 = true
	})
	g2.Get("/", func(ctx *Context) {
		handled2 = true
	})
	g3.Get("/", func(ctx *Context) {
		handled3 = true
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/foo/")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(handled1), qt.Commentf("/foo/ not handled"))

	resp, err = a.TestClient().Get("/foo/bar/")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(handled2), qt.Commentf("/foo/bar/ not handled"))

	resp, err = a.TestClient().Get("/foo/baz/")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(handled3), qt.Commentf("/foo/baz/ not handled"))
}

func TestRouterGroupMiddlewares(t *testing.T) {
	var middleware1, middleware2, handled1, handled2 bool

	a := NewTestApp()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware1 = true
			qt.Check(t, qt.IsFalse(middleware2), qt.Commentf("Second middleware should not yet be called"))
			next(ctx)
		}
	})

	a.Get("/", func(ctx *Context) {
		handled1 = true
	})

	g := a.Group("/v1")

	g.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware2 = true
			qt.Check(t, qt.IsTrue(middleware1), qt.Commentf("First middleware should already be called"))
			next(ctx)
		}
	})
	g.Get("", func(ctx *Context) {
		handled2 = true
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(middleware1), qt.Commentf("First middleware should be called"))
	qt.Check(t, qt.IsFalse(middleware2), qt.Commentf("Second middleware should not be called"))
	qt.Check(t, qt.IsTrue(handled1), qt.Commentf("Handler1 should be called"))
	qt.Check(t, qt.IsFalse(handled2), qt.Commentf("Handler2 should not be called"))

	middleware1 = false
	handled1 = false

	resp, err = a.TestClient().Get("/v1")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(middleware1), qt.Commentf("First middleware should be called"))
	qt.Check(t, qt.IsTrue(middleware2), qt.Commentf("Second middleware should be called"))
	qt.Check(t, qt.IsFalse(handled1), qt.Commentf("Handler1 should not be called"))
	qt.Check(t, qt.IsTrue(handled2), qt.Commentf("Handler2 should be called"))
}
