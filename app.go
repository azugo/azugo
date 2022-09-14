package azugo

import (
	"fmt"
	"sync"

	"azugo.io/azugo/config"
	"azugo.io/azugo/internal/radix"

	"azugo.io/core"
	"azugo.io/core/cert"
	"github.com/dgrr/http2"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type App struct {
	*core.App

	// Routing tree
	trees              []*radix.Tree
	treeMutable        bool
	customMethodsIndex map[string]int
	registeredPaths    map[string][]string
	// Router middlewares
	middlewares []RequestHandlerFunc
	// Cached value of global (*) allowed methods
	globalAllowed string
	// Request context pool
	ctxPool sync.Pool

	// Pointer to the originally set base path in RouterOptions
	originalBasePath *string
	// Cached value of base path
	fixedBasePath string
	pathLock      sync.RWMutex

	// Router options
	RouterOptions RouterOptions

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

		trees:              make([]*radix.Tree, 10),
		customMethodsIndex: make(map[string]int),
		registeredPaths:    make(map[string][]string),
		middlewares:        make([]RequestHandlerFunc, 0, 10),

		RouterOptions: RouterOptions{
			Proxy:                  defaultProxyOptions,
			CORS:                   defaultCORSOptions,
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
			PanicHandler: func(ctx *Context, err any) {
				ctx.Log().Error("Unhandled error", zap.Any("error", err))
			},
			GlobalOPTIONS: func(ctx *Context) {
				ctx.StatusCode(fasthttp.StatusNoContent)
			},
		},

		ServerOptions: ServerOptions{
			RequestReadBufferSize:   8192,
			ResponseWriteBufferSize: 8192,
		},

		MetricsOptions: defaultMetricsOptions,
	}
	return a
}

// basePath returns base path of the application
func (a *App) basePath() string {
	a.pathLock.RLock()
	defer a.pathLock.RUnlock()

	if a.originalBasePath == nil || *a.originalBasePath != a.Config().Server.Path {
		a.pathLock.RUnlock()
		a.pathLock.Lock()

		a.originalBasePath = &a.Config().Server.Path
		a.fixedBasePath = a.Config().Server.Path
		// Add leading slash
		if len(a.fixedBasePath) > 0 && a.fixedBasePath[0] != '/' {
			a.fixedBasePath = "/" + a.fixedBasePath
		}
		// Strip trailing slash
		if len(a.fixedBasePath) > 0 && a.fixedBasePath[len(a.fixedBasePath)-1] == '/' {
			a.fixedBasePath = a.fixedBasePath[:len(a.fixedBasePath)-1]
		}

		a.pathLock.Unlock()
		a.pathLock.RLock()
	}
	return a.fixedBasePath
}

// SetConfig binds application configuration to the application
func (a *App) SetConfig(cmd *cobra.Command, conf *config.Configuration) {
	if a.config != nil && a.config.Ready() {
		return
	}

	a.config = conf
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
