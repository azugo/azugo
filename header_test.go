package azugo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestHeaders(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	want := "gopher"

	a.Get("/user", func(ctx *Context) {
		param := ctx.Header.Get("X-USER")
		assert.Equal(t, want, param, "wrong request header value")

		ctx.Header.Set("X-Real-User", param)
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("X-User", want))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, want, string(resp.Header.Peek("X-Real-User")), "wrong response header value")
}

func TestHeaderDel(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		param := ctx.Header.Get("X-USER")
		assert.NotEmpty(t, param, "request header value should not be empty")

		ctx.Header.Del("X-user")

		param = ctx.Header.Get("X-USER")
		assert.Empty(t, param, "request header value should be empty")
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("X-User", "gopher"))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
}

func TestHeaderValues(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		param := ctx.Header.Values("X-Users")

		assert.ElementsMatch(t, []string{"gopher", "user", "test"}, param, "wrong request header values")

		param = ctx.Header.Values("X-User")
		assert.ElementsMatch(t, []string{"gopher"}, param, "wrong request header values")
	})

	c := a.TestClient()
	resp, err := c.Get("/user", c.WithHeader("X-Users", "gopher,user"), c.WithHeader("X-Users", "test"), c.WithHeader("X-User", "gopher"))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
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
	require.NoError(t, err)

	values := make([]string, 0)
	resp.Header.VisitAll(func(key, value []byte) {
		if string(key) == "X-User" {
			values = append(values, string(value))
		}
	})

	assert.ElementsMatch(t, []string{"gopher", "test"}, values, "wrong response header values")
}
