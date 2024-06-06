package middleware

import (
	"testing"

	"azugo.io/azugo"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestCORSHandlerOptions(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS(a.RouterOptions().CORS.
		SetMethods("GET", "POST").
		SetOrigins("http://1.0.1.0").
		SetHeaders("X-Test-Header")),
	)

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("Hello, world!")
	})

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()
	resp, err := c.Get("/test", c.WithHeader("Origin", "http://1.0.1.0"))
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)

	methodsHeader := string(resp.Header.Peek("Access-Control-Allow-Methods"))
	qt.Check(t, qt.Equals(methodsHeader, "GET, POST"))

	originHeader := string(resp.Header.Peek("Access-Control-Allow-Origin"))
	qt.Check(t, qt.Equals(originHeader, "http://1.0.1.0"))

	headersHeader := string(resp.Header.Peek("Access-Control-Allow-Headers"))
	qt.Check(t, qt.Equals(headersHeader, "X-Test-Header"))
}

func TestCORSHandlerNotAllowed(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS(&a.RouterOptions().CORS))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("Hello, world!")
	})

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()
	resp, err := c.Get("/test", c.WithHeader("Origin", "http://1.0.1.0"))
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)

	methodsHeader := string(resp.Header.Peek("Access-Control-Allow-Methods"))
	qt.Check(t, qt.Equals(methodsHeader, ""))

	originHeader := string(resp.Header.Peek("Access-Control-Allow-Origin"))
	qt.Check(t, qt.Equals(originHeader, ""))

	headersHeader := string(resp.Header.Peek("Access-Control-Allow-Headers"))
	qt.Check(t, qt.Equals(headersHeader, ""))
}

func TestCORSHandlerAllowedOriginsAll(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS(a.RouterOptions().CORS.SetOrigins("*")))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("Hello, world!")
	})

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()
	resp, err := c.Get("/test", c.WithHeader("Origin", "http://1.0.1.0"))
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)

	originHeader := string(resp.Header.Peek("Access-Control-Allow-Origin"))
	qt.Check(t, qt.Equals(originHeader, "http://1.0.1.0"))
}

func TestCORSHandlerOriginDisallowed(t *testing.T) {
	a := azugo.NewTestApp()

	a.Use(CORS(a.RouterOptions().CORS.SetOrigins("http://1.1.1.1")))

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("Hello, world!")
	})

	a.Start(t)
	defer a.Stop()

	c := a.TestClient()
	resp, err := c.Get("/test")
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)

	originHeader := string(resp.Header.Peek("Access-Control-Allow-Origin"))
	qt.Check(t, qt.Equals(originHeader, ""))
}
