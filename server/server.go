package server

import (
	"context"
	"errors"

	"azugo.io/azugo"
	"azugo.io/azugo/config"
	"azugo.io/azugo/middleware"

	"azugo.io/core/server"
	"github.com/spf13/cobra"
)

// Options is a set of options for the server.
type Options server.Options

// newApp creates a new Azugo app with configuration loaded but without any middlewares.
func newApp(cmd *cobra.Command, opt Options) (*azugo.App, error) {
	var conf *config.Configuration

	c := opt.Configuration
	if c == nil {
		conf = config.New()
		c = conf
		opt.Configuration = conf
	}

	if configurable, ok := c.(config.Configurable); ok {
		conf = configurable.ServerCore()
	} else {
		return nil, errors.New("configuration must implement Configurable interface")
	}

	ca, err := server.New(cmd, server.Options(opt))
	if err != nil {
		return nil, err
	}

	a := azugo.New(ca)
	a.SetConfig(cmd, conf)

	return a, nil
}

// New returns new Azugo pre-configured server with default set of middlewares and default router options.
func New(cmd *cobra.Command, opt Options) (*azugo.App, error) {
	a, err := newApp(cmd, opt)
	if err != nil {
		return nil, err
	}

	// Apply configuration.
	a.ApplyConfig()

	// Proxy support for client IP
	a.Use(middleware.RealIP)
	// Log requests
	a.Use(middleware.RequestLogger)
	// Provide metrics
	if a.Config().Metrics.Enabled {
		a.Use(middleware.Metrics(a.Config().Metrics.Path))
	}
	// Support CORS headers
	a.Use(middleware.CORS(&a.RouterOptions().CORS))

	return a, nil
}

// Run starts an application and waits for an interrupt or termination signal
// before stopping it gracefully.
func Run(a server.Runnable) {
	server.Run(a)
}

// RunContext starts an application and waits for the context to be cancelled
// before stopping it gracefully.
func RunContext(ctx context.Context, a server.Runnable) {
	server.RunContext(ctx, a)
}
