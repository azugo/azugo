package azugo

import (
	"net"
	"net/netip"
	"testing"

	"azugo.io/azugo/paginator"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestRequestBasic(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		assert.NotNil(t, ctx.App(), "App should not be nil")
		assert.Equal(t, EnvironmentProduction, ctx.Env(), "Environment should be production")
		assert.Equal(t, fasthttp.MethodGet, ctx.Method(), "Request method should be GET")
		assert.Equal(t, "/user", ctx.Path(), "Request path should be /user")
		assert.Equal(t, "test/1.0", ctx.UserAgent(), "User agent should be test/1.0")
		assert.Nil(t, ctx.IP(), "Client IP address should be nil")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("User-Agent", "test/1.0"))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
}

func TestRequestBaseURLRoot(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		assert.Equal(t, "http://test", ctx.BaseURL(), "BaseURL should be equal to http://test")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
}

func TestRequestBaseURLWithBasePath(t *testing.T) {
	a := NewTestApp()
	a.Config().Server.Path = "/test"
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		assert.Equal(t, "http://test/test", ctx.BaseURL(), "BaseURL should be equal to http://test/test")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/test/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
}

func TestRequestTLSBaseURL(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		assert.Equal(t, "https://test.local", ctx.BaseURL(), "BaseURL should be equal to https://local")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user",
		c.WithHeader("X-Forwarded-Proto", "https"),
		c.WithHeader("X-Forwarded-Host", "test.local"))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
}

func TestRequestCustomHost(t *testing.T) {
	a := NewTestApp()
	a.RouterOptions.Host = "test.local"
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		assert.Equal(t, "http://test.local", ctx.BaseURL(), "BaseURL should be equal to http://test.local")
		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("X-Forwarded-Proto", "http"))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
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
		assert.NotNil(t, ctx.IP(), "Client IP address should not be nil")
		if ctx.IP() != nil {
			assert.Equal(t, "1.1.1.1", ctx.IP().String(), "Client IP should be equal to 1.1.1.1")
		}

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
}

func TestRequestPaging(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		p := ctx.Paging()
		assert.Equal(t, 2, p.Current(), "Page should be 2")
		assert.Equal(t, 10, p.PageSize(), "Page size should be 10")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithQuery(map[string]any{
		paginator.QueryParameterPage:    2,
		paginator.QueryParameterPerPage: 10,
	}))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
}

func TestRequestDefaultPaging(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		p := ctx.Paging()
		assert.Equal(t, 1, p.Current(), "Page should be 1")
		assert.Equal(t, 20, p.PageSize(), "Page size should be 20")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
}
