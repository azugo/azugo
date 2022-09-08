package server

import (
	"errors"

	"azugo.io/azugo"
	"azugo.io/azugo/config"
	"azugo.io/azugo/middleware"

	"azugo.io/core/server"
	"github.com/spf13/cobra"
)

// ServerOptions is a set of options for the server.
type ServerOptions server.ServerOptions

// New returns new Azugo pre-configured server with default set of middlewares and default router options.
func New(cmd *cobra.Command, opt ServerOptions) (*azugo.App, error) {
	a := azugo.New()

	// Support extended configuration.
	var conf *config.Configuration
	c := opt.Configuration
	if c == nil {
		conf = config.New()
		c = conf
		opt.Configuration = conf
	} else if configurable, ok := c.(config.Configurable); ok {
		conf = configurable.ServerCore()
	} else {
		return nil, errors.New("configuration must implement Configurable interface")
	}
	a.SetConfig(cmd, conf)

	ca, err := server.New(cmd, server.ServerOptions(opt))
	if err != nil {
		return nil, err
	}
	a.App = ca

	// Apply configuration.
	applyConfig(a)

	// Proxy support for client IP
	a.Use(middleware.RealIP)
	// Log requests
	a.Use(middleware.RequestLogger(a.Log().Named("http")))
	// Provide metrics
	if a.Config().Metrics.Enabled {
		a.Use(middleware.Metrics(a.Config().Metrics.Path))
	}
	// Support CORS headers
	a.Use(middleware.CORS(&a.RouterOptions.CORS))

	return a, nil
}

func applyConfig(a *azugo.App) {
	conf := a.Config()

	// Apply CORS configuration.
	if len(conf.CORS.Origins) > 0 {
		a.RouterOptions.CORS.SetOrigins(conf.CORS.Origins...)
	}
	// Apply Proxy configuration.
	a.RouterOptions.Proxy.Clear().ForwardLimit = conf.Proxy.Limit
	for _, p := range conf.Proxy.Address {
		a.RouterOptions.Proxy.Add(p)
	}
	// Apply Metrics configuration.
	if conf.Metrics.Enabled {
		a.MetricsOptions.Clear()
		for _, p := range conf.Metrics.Address {
			a.MetricsOptions.Add(p)
		}
	}
}

// Run starts an application and waits for it to finish
func Run(a server.Runnable) {
	server.Run(a)
}
