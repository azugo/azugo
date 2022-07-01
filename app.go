package azugo

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"azugo.io/azugo/cache"
	"azugo.io/azugo/cert"
	"azugo.io/azugo/config"
	"azugo.io/azugo/internal/radix"
	"azugo.io/azugo/validation"

	"github.com/dgrr/http2"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type App struct {
	env Environment

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

	// Validate instance
	validate *validation.Validate

	// Router options
	RouterOptions RouterOptions

	// Logger
	logger *zap.Logger

	// Configuration
	config *config.Configuration

	// Cache
	cache *cache.Cache

	// Background context
	bgctx  context.Context
	bgstop context.CancelFunc

	// App settings
	AppVer       string
	AppBuiltWith string
	AppName      string

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
	ctx, stop := context.WithCancel(context.Background())

	a := &App{
		env: NewEnvironment(EnvironmentProduction),

		bgctx:  ctx,
		bgstop: stop,

		trees:              make([]*radix.Tree, 10),
		customMethodsIndex: make(map[string]int),
		registeredPaths:    make(map[string][]string),
		middlewares:        make([]RequestHandlerFunc, 0, 10),

		validate: validation.New(),

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
		},

		ServerOptions: ServerOptions{
			RequestReadBufferSize:   8192,
			ResponseWriteBufferSize: 8192,
		},

		MetricsOptions: defaultMetricsOptions,
	}
	return a
}

// SetVersion sets application version and built with tags
func (a *App) SetVersion(version, builtWith string) {
	a.AppVer = version
	a.AppBuiltWith = builtWith
}

// Env returns the current application environment
func (a *App) Env() Environment {
	return a.env
}

// Validate returns validation service instance.
func (a *App) Validate() *validation.Validate {
	return a.validate
}

// BackgroundContext returns global background context
func (a *App) BackgroundContext() context.Context {
	return a.bgctx
}

func (a *App) String() string {
	name := a.AppName
	if len(name) == 0 {
		name = "Azugo"
	}

	bw := a.AppBuiltWith
	if len(bw) > 0 {
		bw = fmt.Sprintf(" (built with %s)", bw)
	}
	return fmt.Sprintf("%s %s%s", name, a.AppVer, bw)
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
	if cmd != nil {
		a.config.BindCmd(cmd)
	}
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
	if err := a.initLogger(); err != nil {
		return err
	}
	if err := a.initCache(); err != nil {
		return err
	}

	config := a.Config().Server

	a.Log().Info(fmt.Sprintf("Starting %s...", a.String()))

	server := &fasthttp.Server{
		NoDefaultServerHeader:        true,
		Handler:                      a.Handler,
		Logger:                       zap.NewStdLog(a.Log().Named("http")),
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
		ReadBufferSize:               a.ServerOptions.RequestReadBufferSize,
		WriteBufferSize:              a.ServerOptions.ResponseWriteBufferSize,
	}

	// HTTP2 is supported only over HTTPS
	if config.HTTPS != nil && config.HTTPS.Enabled {
		http2.ConfigureServer(server, http2.ServerConfig{})
	}

	var wg sync.WaitGroup
	if config.HTTP != nil && config.HTTP.Enabled {
		addr := config.HTTP.Address
		if addr == "0.0.0.0" {
			addr = ""
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			a.Log().Info(fmt.Sprintf("Listening on http://%s:%d%s...", config.HTTP.Address, config.HTTP.Port, config.Path))
			if err := server.ListenAndServe(fmt.Sprintf("%s:%d", addr, config.HTTP.Port)); err != nil {
				a.Log().Error("failed to start HTTP server", zap.Error(err))
			}
		}()
	}

	if config.HTTPS != nil && config.HTTPS.Enabled {
		addr := config.HTTPS.Address
		if addr == "0.0.0.0" {
			addr = ""
		}

		var certData []byte
		var keyData []byte
		var err error
		if len(config.HTTPS.CertificatePEMFile) > 0 {
			certData, keyData, err = cert.LoadTLSCertificate(config.HTTPS.CertificatePEMFile)
			if err != nil {
				a.Log().Error("failed to load TLS certificate", zap.Error(err))
				return err
			}
		} else {
			certData, keyData, err = cert.DevTLSCertificate("azugo", "localhost")
			if err != nil {
				a.Log().Error("failed to load or generate self-signed TLS certificate", zap.Error(err))
				return err
			}
		}

		wg.Add(1)

		go func() {
			defer wg.Done()
			a.Log().Info(fmt.Sprintf("Listening on https://%s:%d%s...", config.HTTPS.Address, config.HTTPS.Port, config.Path))
			if err := server.ListenAndServeTLSEmbed(fmt.Sprintf("%s:%d", addr, config.HTTPS.Port), certData, keyData); err != nil {
				a.Log().Error("failed to start HTTPS server", zap.Error(err))
			}
		}()
	}

	wg.Wait()

	return nil
}

// Stop application and its services
func (a *App) Stop() {
	a.bgstop()

	a.closeCache()
}

// Runnable provides methods to run application that will gracefully stop
type Runnable interface {
	Start() error
	Log() *zap.Logger
	Stop()
}

// Run starts an application and waits for it to finish
func Run(a Runnable) {
	// Catch interrupts for gracefully stopping background node proecess
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := a.Start(); err != nil {
			a.Log().With(zap.Error(err)).Fatal("failed to start service")
		}
	}()

	<-done

	a.Stop()
}
