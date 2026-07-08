// Package azugo is a fast and simple web framework for building APIs.
package azugo

import (
	"context"
	"crypto/rand"
	"fmt"
	"sync"
	"time"

	"azugo.io/azugo/config"

	"azugo.io/core"
	"azugo.io/core/cert"
	"azugo.io/core/http"
	"github.com/lafriks/http2"
	"github.com/oklog/ulid/v2"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// App is the main application instance.
type App struct {
	noCopy noCopy

	*core.App

	router     RouteSwitcher
	defaultMux *mux
	entropy    ulid.MonotonicReader

	// Request context pool
	ctxPool sync.Pool
	ctxExt  ExtendedContext

	// Configuration
	config *config.Configuration

	// HTTP client
	http     http.Client
	httpOpts []http.Option
	httpSync sync.RWMutex

	// Metrics options
	MetricsOptions MetricsOptions

	// Healthz options
	HealthzOptions TrustedSource

	// Server options
	ServerOptions ServerOptions

	// Running servers
	serverLock sync.Mutex
	server     *fasthttp.Server
	h2server   *http2.Server
}

// ServerOptions configures the HTTP server buffer sizes.
type ServerOptions struct {
	// Per-connection buffer size for requests' reading.
	// This also limits the maximum header size.
	//
	// Increase this buffer if your clients send multi-KB RequestURIs
	// and/or multi-KB headers (for example, BIG cookies).
	//
	// Default buffer size 8K is used if not set.
	RequestReadBufferSize int

	// Per-connection buffer size for responses' writing.
	//
	// Default buffer size 8K is used if not set.
	ResponseWriteBufferSize int
}

// New creates a new Azugo application.
func New(opts ...*core.App) *App {
	var app *core.App
	if len(opts) > 0 {
		app = opts[0]
	} else {
		app = core.New()
	}

	a := &App{
		App: app,
		entropy: &ulid.LockedMonotonicReader{
			MonotonicReader: ulid.Monotonic(rand.Reader, 0),
		},

		ServerOptions: ServerOptions{
			RequestReadBufferSize:   8192,
			ResponseWriteBufferSize: 8192,
		},

		MetricsOptions: defaultMetricsOptions,
		HealthzOptions: defaultHealthzTrustedSource,
	}

	a.defaultMux = newMux(a)
	a.router = defaultRouter{App: a}

	return a
}

// RouterOptions for default router.
func (a *App) RouterOptions() *RouterOptions {
	return a.defaultMux.RouterOptions
}

// SetConfig binds application configuration to the application.
func (a *App) SetConfig(_ *cobra.Command, conf *config.Configuration) {
	if a.config != nil && a.config.Ready() {
		return
	}

	a.config = conf
}

// ApplyConfig applies the loaded configuration to the application.
func (a *App) ApplyConfig() {
	conf := a.Config()

	// Apply configuration to default server router options.
	a.RouterOptions().ApplyConfig(conf)

	// Apply Metrics configuration.
	if conf.Metrics.Enabled {
		a.MetricsOptions.Clear()

		for _, p := range conf.Metrics.Address {
			a.MetricsOptions.Add(p)
		}
	}

	// Apply Healthz configuration.
	if conf.Healthz.Enabled {
		a.HealthzOptions.Clear()

		for _, p := range conf.Healthz.Address {
			a.HealthzOptions.Add(p)
		}
	}
}

// SetRouterSwitch sets router switcher that selects router based on request context.
func (a *App) SetRouterSwitch(r RouteSwitcher) {
	if r == nil {
		a.router = defaultRouter{App: a}

		return
	}

	a.router = customRouter{App: a, custom: r}
}

// SetExtendedContext sets the context extension.
//
// Deprecated: use Context.SetContext from a handler or middleware to install the
// effective request context instead. See ExtendedContext for details.
func (a *App) SetExtendedContext(ext ExtendedContext) {
	a.ctxExt = ext
}

// Config returns application configuration.
//
// Panics if configuration is not loaded.
func (a *App) Config() *config.Configuration {
	if a.config == nil || !a.config.Ready() {
		panic("configuration is not loaded")
	}

	return a.config
}

// Start web application.
func (a *App) Start() error {
	if err := a.App.Start(); err != nil {
		return err
	}

	conf := a.Config().Server

	server := &fasthttp.Server{
		NoDefaultServerHeader:        true,
		Handler:                      a.Handler,
		Logger:                       zap.NewStdLog(a.Log().Named("http")),
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
		ReadBufferSize:               a.ServerOptions.RequestReadBufferSize,
		WriteBufferSize:              a.ServerOptions.ResponseWriteBufferSize,
		ReadTimeout:                  conf.ReadTimeout,
		WriteTimeout:                 conf.WriteTimeout,
		IdleTimeout:                  conf.IdleTimeout,
		MaxRequestBodySize:           conf.MaxRequestBodySize,
	}

	var h2server *http2.Server

	// HTTP2 is supported only over HTTPS
	if conf.HTTPS != nil && conf.HTTPS.Enabled {
		h2server = http2.ConfigureServer(server, http2.ServerConfig{
			PingInterval:         30 * time.Second,
			MaxConcurrentStreams: 256,
		})
	}

	a.serverLock.Lock()
	a.server = server
	a.h2server = h2server
	a.serverLock.Unlock()

	var wg sync.WaitGroup

	if conf.HTTP != nil && conf.HTTP.Enabled {
		addr := conf.HTTP.Address
		if addr == "0.0.0.0" {
			addr = ""
		}

		wg.Go(func() {
			a.Log().Info(fmt.Sprintf("Listening on http://%s:%d%s...", conf.HTTP.Address, conf.HTTP.Port, conf.Path))

			if err := server.ListenAndServe(fmt.Sprintf("%s:%d", addr, conf.HTTP.Port)); err != nil {
				a.Log().Error("failed to start HTTP server", zap.Error(err))
			}
		})
	}

	if conf.HTTPS != nil && conf.HTTPS.Enabled {
		addr := conf.HTTPS.Address
		if addr == "0.0.0.0" {
			addr = ""
		}

		var (
			certData, keyData []byte

			err error
		)

		if len(conf.HTTPS.CertificatePEMFile) > 0 {
			certData, keyData, err = cert.LoadPEMFromFile(conf.HTTPS.CertificatePEMFile)
			if err != nil {
				a.Log().Error("failed to load TLS certificate", zap.Error(err))

				return err
			}
		} else {
			certData, keyData, err = cert.DevPEMFile("azugo", "localhost")
			if err != nil {
				a.Log().Error("failed to load or generate self-signed TLS certificate", zap.Error(err))

				return err
			}
		}

		wg.Go(func() {
			a.Log().Info(fmt.Sprintf("Listening on https://%s:%d%s...", conf.HTTPS.Address, conf.HTTPS.Port, conf.Path))

			if err := server.ListenAndServeTLSEmbed(fmt.Sprintf("%s:%d", addr, conf.HTTPS.Port), certData, keyData); err != nil {
				a.Log().Error("failed to start HTTPS server", zap.Error(err))
			}
		})
	}

	wg.Wait()

	return nil
}

// Stop web application and its services waiting for active connections to finish.
func (a *App) Stop() {
	a.serverLock.Lock()
	server, h2server := a.server, a.h2server
	a.server, a.h2server = nil, nil
	a.serverLock.Unlock()

	if server != nil {
		ctx, cancel := context.WithTimeout(context.Background(), a.Config().Server.ShutdownTimeout)
		defer cancel()

		if h2server != nil {
			if err := h2server.Shutdown(ctx); err != nil {
				a.Log().Warn("failed to gracefully shut down HTTP2 connections", zap.Error(err))
			}
		}

		if err := server.ShutdownWithContext(ctx); err != nil {
			a.Log().Warn("failed to gracefully shut down HTTP server", zap.Error(err))
		}
	}

	a.App.Stop()
}
