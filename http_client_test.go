package azugo

import (
	"context"
	"testing"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
)

func TestHTTPClient_Context(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Instrumentation(func(ctx context.Context, op string, args ...any) func(err error) {
		if op != http.InstrumentationRequest {
			return func(_ error) {}
		}

		c, ok := ctx.(*Context)

		qt.Check(t, qt.IsTrue(ok), qt.Commentf("expected *Context, got %T", ctx))
		qt.Check(t, qt.IsNotNil(c), qt.Commentf("expected non-nil *Context"))

		return func(_ error) {}
	})

	a.Get("/test", func(ctx *Context) {
		_, err := ctx.HTTPClient().Get("http://example.com")
		if err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(200)
	})

	_, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
}

func TestHTTPClient_SimpleTracing(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	expectedTraceparent := "00-0af7651916cd43dd8448eb211c80319c-b7ad6b7169203331-01"
	forwardedTraceparent := ""

	a.App.AddHTTPClientOption(http.RequestFunc(func(ctx context.Context, req *http.Request) error {
		c, ok := ctx.(*Context)
		if !ok {
			return nil
		}

		tpar, ok := c.UserValue("traceparent").(string)
		if ok && tpar != "" {
			req.Header.Set("Traceparent", tpar)
		}

		return nil
	}))

	a.App.Instrumentation(func(_ context.Context, op string, args ...any) func(err error) {
		req, _, ok := http.InstrRequest(op, args...)
		if !ok {
			return func(_ error) {}
		}

		forwardedTraceparent = string(req.Header.Peek("Traceparent"))

		return func(_ error) {}
	})

	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			ctx.SetUserValue("traceparent", expectedTraceparent)
			next(ctx)
		}
	})

	a.Get("/test", func(ctx *Context) {
		_, err := ctx.HTTPClient().Get("http://example.com")
		if err != nil {
			ctx.Error(err)
			return
		}

		ctx.StatusCode(200)
	})

	_, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(forwardedTraceparent, expectedTraceparent))
}

func TestContext_ImplementsHTTPClientProvider(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	x := func(ctx context.Context) {
		if !qt.Check(t, qt.Implements[http.ClientProvider](ctx)) {
			return
		}

		provider, ok := ctx.(http.ClientProvider)
		qt.Check(t, qt.IsTrue(ok), qt.Commentf("expected http.ClientProvider, got %T", ctx))

		client := provider.HTTPClient()
		qt.Check(t, qt.IsNotNil(client), qt.Commentf("expected non-nil *http.Client"))
	}

	a.Get("/test", func(ctx *Context) {
		x(ctx)
	})

	_, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
}

func TestApp_ImplementsHTTPClientProvider(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	qt.Check(t, qt.Implements[http.ClientProvider](a))
}
