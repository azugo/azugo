package main

import (
	"azugo.io/azugo"
	"azugo.io/azugo/middleware"

	"github.com/valyala/fasthttp"
)

func main() {
	a := azugo.New()
	a.AppName = "REST API Example"

	a.Use(middleware.RealIP)
	a.Use(middleware.RequestLogger(a.Log().Named("http")))

	a.Get("/", func(ctx *azugo.Context) {
		ctx.StatusCode(fasthttp.StatusOK).Text("Hello, world!")
	})
	if err := a.Start(); err != nil {
		panic(err)
	}
}
