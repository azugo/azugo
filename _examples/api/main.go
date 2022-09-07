package main

import (
	"net/url"

	"azugo.io/azugo"
	"azugo.io/azugo/config"
	"azugo.io/azugo/server"
	cs "azugo.io/core/server"
	"github.com/valyala/fasthttp"
)

// Configuration represents application configuration
type Configuration struct {
	*config.Configuration `mapstructure:",squash"`

	// Custom configuration section.
	Custom string `mapstructure:"custom"`
}

type TestRequest struct {
	Name string `json:"name" validate:"required,max=50"`
}

func main() {
	conf := &Configuration{
		Configuration: config.New(),
	}

	a, err := server.New(nil, cs.ServerOptions{
		AppName: "REST API Example",

		Configuration: conf,
	})
	if err != nil {
		panic(err)
	}

	a.Get("/hello", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK).Text("Hello, world!")
	})
	a.Post("/test", func(ctx *azugo.Context) {
		req := &TestRequest{}
		if err := ctx.Body.JSON(req); err != nil {
			ctx.Error(err)
			return
		}
		ctx.JSON(struct {
			ID int `json:"id"`
		}{1})
	})

	u, err := url.Parse("https://example.com/")
	if err != nil {
		panic(err)
	}
	a.Proxy("/example", azugo.ProxyUpstream(u))

	cs.Run(a)
}
