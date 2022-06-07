package middleware

import (
	"testing"

	"azugo.io/azugo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestCORSHandlerOptions(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS(a.RouterOptions.CORS.
		SetMethods("GET", "POST").
		SetOrigins("http://1.0.1.0").
		SetHeaders("X-Test-Header")),
	)

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK).Text("Hello, world!")
	})

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()
	resp, err := c.Get("/test", c.WithHeader("Origin", "http://1.0.1.0"))
	require.NoError(t, err)
	defer fasthttp.ReleaseResponse(resp)

	methodsHeader := string(resp.Header.Peek("Access-Control-Allow-Methods"))
	assert.Equal(t, "GET, POST", methodsHeader)

	originHeader := string(resp.Header.Peek("Access-Control-Allow-Origin"))
	assert.Equal(t, "http://1.0.1.0", originHeader)

	headersHeader := string(resp.Header.Peek("Access-Control-Allow-Headers"))
	assert.Equal(t, "X-Test-Header", headersHeader)
}

func TestCORSHandlerNotAllowed(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS(&a.RouterOptions.CORS))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK).Text("Hello, world!")
	})

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()
	resp, err := c.Get("/test", c.WithHeader("Origin", "http://1.0.1.0"))
	require.NoError(t, err)
	defer fasthttp.ReleaseResponse(resp)

	methodsHeader := string(resp.Header.Peek("Access-Control-Allow-Methods"))
	assert.Equal(t, "", methodsHeader)

	originHeader := string(resp.Header.Peek("Access-Control-Allow-Origin"))
	assert.Equal(t, "", originHeader)

	headersHeader := string(resp.Header.Peek("Access-Control-Allow-Headers"))
	assert.Equal(t, "", headersHeader)
}

func TestCORSHandlerAllowedOriginsAll(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS(a.RouterOptions.CORS.SetOrigins("*")))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK).Text("Hello, world!")
	})

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()
	resp, err := c.Get("/test", c.WithHeader("Origin", "http://1.0.1.0"))
	require.NoError(t, err)
	defer fasthttp.ReleaseResponse(resp)

	originHeader := string(resp.Header.Peek("Access-Control-Allow-Origin"))
	assert.Equal(t, "http://1.0.1.0", originHeader)
}

func TestCORSHandlerOriginDisallowed(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS(a.RouterOptions.CORS.SetOrigins("http://1.1.1.1")))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK).Text("Hello, world!")
	})

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()
	resp, err := c.Get("/test")
	require.NoError(t, err)
	defer fasthttp.ReleaseResponse(resp)

	originHeader := string(resp.Header.Peek("Access-Control-Allow-Origin"))
	assert.Equal(t, "", originHeader)
}
