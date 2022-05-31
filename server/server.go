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
	Configuration interface{}
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

	a.Use(middleware.RealIP)
	a.Use(middleware.RequestLogger(a.Log().Named("http")))
	a.Use(middleware.Metrics(azugo.DefaultMetricPath))

	return a, nil
}
