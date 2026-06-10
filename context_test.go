package azugo

import (
	"context"
	"testing"
	"time"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestImplementsContextInterface(t *testing.T) {
	qt.Check(t, qt.Implements[context.Context](&Context{}))
}

type testExtValueContext struct{}

func (t *testExtValueContext) Context(ctx context.Context) context.Context {
	return context.WithValue(ctx, "test", "value")
}

type testExtDeadlineContext struct {
	cancel context.CancelFunc
}

func (t *testExtDeadlineContext) Context(ctx context.Context) context.Context {
	if t.cancel != nil {
		t.cancel()
	}

	c, cancel := context.WithDeadline(context.TODO(), time.Now().Add(time.Minute))
	t.cancel = cancel

	return c
}

func TestContextValueExtension(t *testing.T) {
	app := NewTestApp()

	app.SetExtendedContext(&testExtValueContext{})

	app.Start(t)
	defer app.Stop()

	app.Get("/test", func(ctx *Context) {
		v := ctx.Value("test")

		if v == "value" {
			ctx.StatusCode(200)
			return
		}

		ctx.StatusCode(500)
	})

	resp, err := app.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
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

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}
