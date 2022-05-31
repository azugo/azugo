package main

import (
	"net/url"

	"azugo.io/azugo"
	"azugo.io/azugo/server"

	"github.com/valyala/fasthttp"
)

type TestRequest struct {
	Name string `json:"name" validate:"required,max=50"`
}

func main() {
	a, err := server.New(nil, server.ServerOptions{
		AppName: "REST API Example",
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

	azugo.Run(a)
}
