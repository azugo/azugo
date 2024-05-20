package azugo

import (
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestHeaders(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	want := "gopher"

	a.Get("/user", func(ctx *Context) {
		param := ctx.Header.Get("X-USER")
		qt.Check(t, qt.Equals(param, want), qt.Commentf("wrong request header value"))

		ctx.Header.Set("X-Real-User", param)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("X-User", want))
	defer fasthttp.ReleaseResponse(resp)

	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.DeepEquals(resp.Header.Peek("X-Real-User"), []byte(want)), qt.Commentf("wrong response header value"))
}

func TestHeaderDel(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		param := ctx.Header.Get("X-USER")
		qt.Check(t, qt.Not(qt.HasLen(param, 0)), qt.Commentf("request header value should not be empty"))

		ctx.Header.Del("X-user")

		param = ctx.Header.Get("X-USER")
		qt.Check(t, qt.HasLen(param, 0), qt.Commentf("request header value should be empty"))
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("X-User", "gopher"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestHeaderValues(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		param := ctx.Header.Values("X-Users")

		qt.Check(t, qt.ContentEquals(param, []string{"gopher", "user", "test"}), qt.Commentf("wrong request header values"))

		param = ctx.Header.Values("X-User")

		qt.Check(t, qt.ContentEquals(param, []string{"gopher"}), qt.Commentf("wrong request header values"))
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("X-Users", "gopher,user"), c.WithHeader("X-Users", "test"), c.WithHeader("X-User", "gopher"))
	defer fasthttp.ReleaseResponse(resp)

	qt.Assert(t, qt.IsNil(err))
}

func TestHeaderAdd(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Header.Set("X-User", "gopher")
		ctx.Header.Add("X-User", "test")
	})

	resp, err := a.TestClient().Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	values := make([]string, 0)
	resp.Header.VisitAll(func(key, value []byte) {
		if string(key) == "X-User" {
			values = append(values, string(value))
		}
	})

	qt.Assert(t, qt.ContentEquals(values, []string{"gopher", "test"}), qt.Commentf("wrong response header values"))
}
