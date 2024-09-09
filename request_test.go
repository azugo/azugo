package azugo

import (
	"net"
	"net/netip"
	"testing"

	"azugo.io/core"
	"azugo.io/core/paginator"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestRequestBasic(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		qt.Check(t, qt.IsNotNil(ctx.App()))
		qt.Check(t, qt.Equals(ctx.Env(), core.EnvironmentProduction))
		qt.Check(t, qt.Equals(ctx.Method(), fasthttp.MethodGet))
		qt.Check(t, qt.Equals(ctx.Path(), "/user"))
		qt.Check(t, qt.Equals(ctx.UserAgent(), "test/1.0"))
		qt.Check(t, qt.IsNil(ctx.IP()))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("User-Agent", "test/1.0"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestBaseURLRoot(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		qt.Check(t, qt.Equals(ctx.BaseURL(), "http://test"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestBaseURLWithBasePath(t *testing.T) {
	a := NewTestApp()
	a.Config().Server.Path = "/test"
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		qt.Check(t, qt.Equals(ctx.BaseURL(), "http://test/test"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/test/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestTLSBaseURL(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		qt.Check(t, qt.Equals(ctx.BaseURL(), "https://test.local"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user",
		c.WithHeader("X-Forwarded-Proto", "https"),
		c.WithHeader("X-Forwarded-Host", "test.local"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestCustomHost(t *testing.T) {
	a := NewTestApp()
	a.RouterOptions().Host = "test.local"
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		qt.Check(t, qt.Equals(ctx.BaseURL(), "http://test.local"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("X-Forwarded-Proto", "http"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestIP(t *testing.T) {
	a := NewTestApp()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			ctx.Context().SetRemoteAddr(net.TCPAddrFromAddrPort(netip.MustParseAddrPort("1.1.1.1:30003")))
			next(ctx)
		}
	})
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		if qt.Check(t, qt.IsNotNil(ctx.IP())) {
			qt.Check(t, qt.Equals(ctx.IP().String(), "1.1.1.1"))
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestPaging(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		p := ctx.Paging()
		qt.Check(t, qt.Equals(p.Current(), 2))
		qt.Check(t, qt.Equals(p.PageSize(), 10))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		paginator.QueryParameterPage:    2,
		paginator.QueryParameterPerPage: 10,
	}))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestDefaultPaging(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		p := ctx.Paging()
		qt.Check(t, qt.Equals(p.Current(), 1))
		qt.Check(t, qt.Equals(p.PageSize(), 20))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestPagingDefaultMaxPageSize(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		p := ctx.Paging()
		qt.Check(t, qt.Equals(p.Current(), 2))
		qt.Check(t, qt.Equals(p.PageSize(), 100))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		paginator.QueryParameterPage:    2,
		paginator.QueryParameterPerPage: 10000,
	}))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestPagingCustomMaxPageSize(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			ctx.SetMaxPageSize(1000)
			next(ctx)
		}
	})

	a.Get("/user", func(ctx *Context) {
		p := ctx.Paging()
		qt.Check(t, qt.Equals(p.Current(), 2))
		qt.Check(t, qt.Equals(p.PageSize(), 1000))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		paginator.QueryParameterPage:    2,
		paginator.QueryParameterPerPage: 10000,
	}))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestAccepts(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/test", func(ctx *Context) {
		qt.Check(t, qt.IsTrue(ctx.Accepts("application/json")))
		qt.Check(t, qt.IsTrue(ctx.Accepts("text/html")))
		qt.Check(t, qt.IsTrue(ctx.Accepts("application/xml")))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/test", c.WithHeader("Accept", "application/json, text/*; q=0.5, */*; q=0.1"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}

func TestRequestAcceptsExplicit(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/test", func(ctx *Context) {
		qt.Check(t, qt.IsTrue(ctx.AcceptsExplicit("application/json")))
		qt.Check(t, qt.IsTrue(ctx.AcceptsExplicit("text/html")))
		qt.Check(t, qt.IsFalse(ctx.AcceptsExplicit("application/xml")))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/test", c.WithHeader("Accept", "application/json, text/*; q=0.5, */*; q=0.1"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
}
