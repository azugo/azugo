package azugo

import (
	"context"
	"testing"
	"time"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestImplementsContextInterface(t *testing.T) {
	qt.Check(t, qt.Implements[context.Context](&Context{}))
}

type testExtValueContext struct{}

func (t *testExtValueContext) Context(ctx context.Context) context.Context {
	return context.WithValue(RequestContext(ctx).Context(), "test", "value")
}

type testExtDeadlineContext struct {
	cancel context.CancelFunc
}

func (t *testExtDeadlineContext) Context(ctx context.Context) context.Context {
	if t.cancel != nil {
		t.cancel()
	}

	c, cancel := context.WithDeadline(context.Background(), time.Now().Add(time.Minute))
	t.cancel = cancel

	return c
}

// TestContextValueExtension covers the deprecated ExtendedContext hook path.
func TestContextValueExtension(t *testing.T) {
	app := NewTestApp()

	app.SetExtendedContext(&testExtValueContext{})

	app.Start(t)
	defer app.Stop()

	app.Get("/test", func(ctx *Context) {
		qt.Check(t, qt.IsNil(ctx.Value("missing")))

		v := ctx.Value("test")

		if v == "value" {
			ctx.StatusCode(200)
			return
		}

		ctx.StatusCode(500)
	})

	resp, err := app.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
}

type testTxKeyType struct{}

var testTxKey testTxKeyType

type testTxContext struct {
	context.Context

	parent context.Context
}

func (c *testTxContext) RequestContext() context.Context { return c.parent }

func wrapTestTx(ctx context.Context) *testTxContext {
	t := &testTxContext{parent: ctx}
	t.Context = context.WithValue(ctx, testTxKey, t)

	return t
}

func TestContextSetContext(t *testing.T) {
	app := NewTestApp()
	app.Start(t)
	defer app.Stop()

	type pushKey struct{}

	deadline := time.Now().Add(time.Minute)

	app.Get("/test", func(ctx *Context) {
		ctx.SetUserValue("base-key", "base-val")

		ctx.SetContext(context.WithValue(ctx.Context(), pushKey{}, "push-val"))
		qt.Check(t, qt.Equals(ctx.Value(pushKey{}).(string), "push-val"))
		qt.Check(t, qt.Equals(ctx.Value("base-key").(string), "base-val"))
		qt.Check(t, qt.IsNil(ctx.Value("missing")))

		dctx, cancel := context.WithDeadline(context.Background(), deadline)
		defer cancel()
		ctx.SetContext(dctx)
		d, ok := ctx.Deadline()
		qt.Check(t, qt.IsTrue(ok))
		qt.Check(t, qt.IsTrue(d.Equal(deadline)))

		var reset context.Context
		ctx.SetContext(reset)
		qt.Check(t, qt.IsNil(ctx.Value(pushKey{})))
		qt.Check(t, qt.Equals(ctx.Value("base-key").(string), "base-val"))

		ctx.StatusCode(http.StatusNoContent)
	})

	resp, err := app.TestClient().Get("/test")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusNoContent))
}

func TestRequestContextRecovery(t *testing.T) {
	app := NewTestApp()
	app.Start(t)
	defer app.Stop()

	type slowKey struct{}

	app.Get("/test", func(ctx *Context) {
		qt.Check(t, qt.Equals(RequestContext(ctx), ctx))
		tx := wrapTestTx(ctx)
		qt.Check(t, qt.Equals(RequestContext(tx), ctx))
		qt.Check(t, qt.Equals(RequestContext(context.WithValue(tx, slowKey{}, "5s")), ctx))
		qt.Check(t, qt.Equals(RequestContext(context.WithValue(ctx, slowKey{}, "x")), ctx))
		qt.Check(t, qt.IsNil(RequestContext(context.Background())))
		var noContext context.Context
		qt.Check(t, qt.IsNil(RequestContext(noContext)))

		ctx.StatusCode(http.StatusNoContent)
	})

	resp, err := app.TestClient().Get("/test")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusNoContent))
}

func TestContextTransactionAndSpanStack(t *testing.T) {
	app := NewTestApp()
	app.Start(t)
	defer app.Stop()

	type spanKey struct{}

	type slowKey struct{}

	app.Get("/test", func(ctx *Context) {
		ctx.SetContext(context.WithValue(ctx.Context(), spanKey{}, "span"))

		tx := wrapTestTx(ctx)
		stack := context.WithValue(tx, slowKey{}, "5s")

		qt.Check(t, qt.Equals(RequestContext(stack), ctx))
		qt.Check(t, qt.Equals(stack.Value(spanKey{}).(string), "span"))
		qt.Check(t, qt.Equals(stack.Value(testTxKey).(*testTxContext), tx))
		qt.Check(t, qt.Equals(stack.Value(slowKey{}).(string), "5s"))

		ctx.StatusCode(http.StatusNoContent)
	})

	resp, err := app.TestClient().Get("/test")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusNoContent))
}

func TestContextDeadlineExtension(t *testing.T) {
	app := NewTestApp()

	ext := &testExtDeadlineContext{}
	app.SetExtendedContext(ext)

	app.Start(t)
	defer app.Stop()
	t.Cleanup(func() {
		if ext.cancel != nil {
			ext.cancel()
		}
	})

	app.Get("/test", func(ctx *Context) {
		_, ok := ctx.Deadline()

		if ok {
			ctx.StatusCode(200)
			return
		}

		ctx.StatusCode(500)
	})

	resp, err := app.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
}
