package azugo

import (
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestRouteValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user/{name}/{id}", func(ctx *Context) {
		name := ctx.Params.String("name")
		qt.Check(t, qt.Equals(name, "gopher"), qt.Commentf("Route parameter name should be equal to gopher"))

		id, err := ctx.Params.Int("id")
		qt.Check(t, qt.IsNil(err), qt.Commentf("Route parameter id should not be nil"))
		qt.Check(t, qt.Equals(id, 1), qt.Commentf("Route parameter id should be equal to 1"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user/gopher/1")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}

func TestRouteInvalidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user/{id}", func(ctx *Context) {
		id, err := ctx.Params.Int64("id")
		qt.Check(t, qt.IsNotNil(err), qt.Commentf("Route parameter name should have error"))
		qt.Check(t, qt.Equals(id, 0), qt.Commentf("Route parameter name should be equal to 0"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user/gopher")
	defer fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}

func TestRouteNonexistingParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user/{id}", func(ctx *Context) {
		qt.Check(t, qt.IsFalse(ctx.Params.Has("type")))
		qt.Check(t, qt.IsTrue(ctx.Params.Has("id")))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user/gopher")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}
