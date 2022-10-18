package azugo

import (
	"fmt"
	"sync"

	"azugo.io/azugo/config"

	"azugo.io/core"
	"azugo.io/core/cert"
	"github.com/dgrr/http2"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type App struct {
	*core.App

	router     RouteSwitcher
	defaultMux *mux

	// Request context pool
	ctxPool sync.Pool

	// Configuration
	config *config.Configuration

	// Metrics options
	MetricsOptions MetricsOptions

	// Server options
	ServerOptions ServerOptions
}

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

func New() *App {
	a := &App{
		App: core.New(),

		ServerOptions: ServerOptions{
			RequestReadBufferSize:   8192,
			ResponseWriteBufferSize: 8192,
		},

		MetricsOptions: defaultMetricsOptions,
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
func (a *App) SetConfig(cmd *cobra.Command, conf *config.Configuration) {
	if a.config != nil && a.config.Ready() {
		return
	}

	a.config = conf
}

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
}

// SetRouterSwitch sets router switcher that selects router based on request context.
func (a *App) SetRouterSwitch(r RouteSwitcher) {
	if r == nil {
		a.router = defaultRouter{App: a}
		return
	}
	a.router = customRouter{App: a, custom: r}
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

	server := &fasthttp.Server{
		NoDefaultServerHeader:        true,
		Handler:                      a.Handler,
		Logger:                       zap.NewStdLog(a.Log().Named("http")),
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
		ReadBufferSize:               a.ServerOptions.RequestReadBufferSize,
		WriteBufferSize:              a.ServerOptions.ResponseWriteBufferSize,
	}

	conf := a.Config().Server

	// HTTP2 is supported only over HTTPS
	if conf.HTTPS != nil && conf.HTTPS.Enabled {
		http2.ConfigureServer(server, http2.ServerConfig{})
	}

	var wg sync.WaitGroup
	if conf.HTTP != nil && conf.HTTP.Enabled {
		addr := conf.HTTP.Address
		if addr == "0.0.0.0" {
			addr = ""
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			a.Log().Info(fmt.Sprintf("Listening on http://%s:%d%s...", conf.HTTP.Address, conf.HTTP.Port, conf.Path))
			if err := server.ListenAndServe(fmt.Sprintf("%s:%d", addr, conf.HTTP.Port)); err != nil {
				a.Log().Error("failed to start HTTP server", zap.Error(err))
			}
		}()
	}

	if conf.HTTPS != nil && conf.HTTPS.Enabled {
		addr := conf.HTTPS.Address
		if addr == "0.0.0.0" {
			addr = ""
		}

		var certData []byte
		var keyData []byte
		var err error
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

		wg.Add(1)

		go func() {
			defer wg.Done()
			a.Log().Info(fmt.Sprintf("Listening on https://%s:%d%s...", conf.HTTPS.Address, conf.HTTPS.Port, conf.Path))
			if err := server.ListenAndServeTLSEmbed(fmt.Sprintf("%s:%d", addr, conf.HTTPS.Port), certData, keyData); err != nil {
				a.Log().Error("failed to start HTTPS server", zap.Error(err))
			}
		}()
	}

	wg.Wait()

	return nil
}
