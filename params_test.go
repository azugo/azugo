package azugo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestRouteValidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user/{name}/{id}", func(ctx *Context) {
		name := ctx.Params.String("name")
		assert.Equal(t, "gopher", name, "Route parameter name should be equal to gopher")

		id, err := ctx.Params.Int("id")
		assert.NoError(t, err, "Route parameter id should not be nil")
		assert.Equal(t, 1, id, "Route parameter id should be equal to 1")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user/gopher/1")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestRouteInvalidParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user/{id}", func(ctx *Context) {
		id, err := ctx.Params.Int64("id")
		assert.Error(t, err, "Route parameter name should have error")
		assert.Equal(t, int64(0), id, "Route parameter name should be equal to 0")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user/gopher")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestRouteNonexistingParams(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user/{id}", func(ctx *Context) {
		assert.False(t, ctx.Params.Has("type"))
		assert.True(t, ctx.Params.Has("id"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/user/gopher")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}
