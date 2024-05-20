package main

import (
	"net/url"

	"azugo.io/azugo"
	"azugo.io/azugo/config"
	"azugo.io/azugo/server"
	"github.com/spf13/viper"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type CustomConfiguration struct {
	Value string `mapstructure:"value"`
}

// Configuration represents application configuration.
type Configuration struct {
	*config.Configuration `mapstructure:",squash"`

	// Custom configuration section.
	Custom *CustomConfiguration `mapstructure:"custom"`
}

func (c *Configuration) ServerCore() *config.Configuration {
	return c.Configuration
}

func (c *Configuration) Bind(prefix string, v *viper.Viper) {
	c.Configuration.Bind(prefix, v)

	c.Custom = config.Bind(c.Custom, "custom", v)
}

type TestRequest struct {
	Name string `json:"name" validate:"required,max=50"`
}

func main() {
	conf := &Configuration{
		Configuration: config.New(),
	}

	a, err := server.New(nil, server.Options{
		AppName: "REST API Example",

		Configuration: conf,
	})
	if err != nil {
		panic(err)
	}

	a.Get("/hello", func(ctx *azugo.Context) {
		ctx.ContentType("application/json")
		ctx.StatusCode(fasthttp.StatusOK)
		ctx.Text("Hello, world!")
	})
	a.Post("/test", func(ctx *azugo.Context) {
		req := &TestRequest{}
		if err := ctx.Body.JSON(req); err != nil {
			ctx.Error(err)

			return
		}

		content, err := ctx.HTTPClient().Get("https://example.com/")
		if err != nil {
			ctx.Error(err)

			return
		}

		ctx.Log().Debug("response", zap.String("content", string(content)))

		ctx.JSON(struct {
			ID int `json:"id"`
		}{1})
	})

	u, err := url.Parse("https://example.com/")
	if err != nil {
		panic(err)
	}

	a.Proxy("/example", azugo.ProxyUpstream(u))

	server.Run(a)
}
