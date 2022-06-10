package server

import (
	"errors"

	"azugo.io/azugo"

	"azugo.io/azugo/config"
	"azugo.io/azugo/middleware"
	"github.com/spf13/cobra"
)

type ServerOptions struct {
	// AppName is the name of the application.
	AppName string
	// AppVer is the version of the application.
	AppVer string

	// Configuration object that implements config.Configurable interface.
	Configuration any
}

// New returns new Azugo pre-configured server with default set of middlewares and default router options.
func New(cmd *cobra.Command, opt ServerOptions) (*azugo.App, error) {
	a := azugo.New()
	a.AppName = opt.AppName
	a.AppVer = opt.AppVer

	// Support extended configuration.
	var conf *config.Configuration
	c := opt.Configuration
	if c == nil {
		conf = config.New()
		c = conf
	} else if configurable, ok := c.(config.Configurable); ok {
		conf = configurable.Core()
	} else {
		return nil, errors.New("configuration must implement Configurable interface")
	}
	a.SetConfig(cmd, conf)

	// Load configuration
	if err := conf.Load(c, string(a.Env())); err != nil {
		return nil, err
	}

	// Apply configuration.
	applyConfig(a)

	// Proxy support for client IP
	a.Use(middleware.RealIP)
	// Log requests
	a.Use(middleware.RequestLogger(a.Log().Named("http")))
	// Provide metrics
	a.Use(middleware.Metrics(azugo.DefaultMetricPath))
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
}
