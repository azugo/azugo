package azugo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestRouterGroupAPI(t *testing.T) {
	var handled, get, head, post, put, patch, delete, connect, options, trace, any bool

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
		any = true
	})
	g.Handle(fasthttp.MethodGet, "/Handler", func(ctx *Context) {
		handled = true
	})

	resp, err := a.TestClient().Get("/v1/GET")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, get, "GET route not handled")

	resp, err = a.TestClient().Head("/v1/HEAD")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, head, "HEAD route not handled")

	resp, err = a.TestClient().Post("/v1/POST", nil)
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, post, "POST route not handled")

	resp, err = a.TestClient().Put("/v1/PUT", nil)
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, put, "PUT route not handled")

	resp, err = a.TestClient().Patch("/v1/PATCH", nil)
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, patch, "PATCH route not handled")

	resp, err = a.TestClient().Delete("/v1/DELETE")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, delete, "DELETE route not handled")

	resp, err = a.TestClient().Connect("/v1/CONNECT")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, connect, "CONNECT route not handled")

	resp, err = a.TestClient().Options("/v1/OPTIONS")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, options, "OPTIONS route not handled")

	resp, err = a.TestClient().Trace("/v1/TRACE")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, trace, "TRACE route not handled")

	resp, err = a.TestClient().Get("/v1/Handler")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, handled, "Handler route not handled")

	for _, method := range httpMethods {
		resp, err = a.TestClient().Call(method, "/v1/ANY", nil)
		fasthttp.ReleaseResponse(resp)
		require.NoError(t, err)
		assert.True(t, any, "ANY route not handled")
		any = false
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
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, handled1, "/foo/ not handled")

	resp, err = a.TestClient().Get("/foo/bar/")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, handled2, "/foo/bar/ not handled")

	resp, err = a.TestClient().Get("/foo/baz/")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, handled3, "/foo/baz/ not handled")
}

func TestRouterGroupMiddlewares(t *testing.T) {
	var middleware1, middleware2, handled1, handled2 bool

	a := NewTestApp()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware1 = true
			assert.False(t, middleware2, "Second middleware should not yet be called")
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
			assert.True(t, middleware1, "First middleware should already be called")
			next(ctx)
		}
	})
	g.Get("", func(ctx *Context) {
		handled2 = true
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, middleware1, "First middleware not be called")
	assert.False(t, middleware2, "Second middleware should not called")
	assert.True(t, handled1, "Handler1 not called")
	assert.False(t, handled2, "Handler2 should not be called")

	middleware1 = false
	handled1 = false

	resp, err = a.TestClient().Get("/v1")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, middleware1, "First middleware not be called")
	assert.True(t, middleware2, "Second middleware should not called")
	assert.False(t, handled1, "Handler1 should not be called")
	assert.True(t, handled2, "Handler2 not called")
}
