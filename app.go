package azugo

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"azugo.io/azugo/internal/radix"
	"azugo.io/azugo/validation"

	"github.com/dgrr/http2"
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

	// Background context
	bgctx  context.Context
	bgstop context.CancelFunc

	// AppVer settings
	AppVer       string
	AppBuiltWith string
	AppName      string

	// Metrics options
	MetricsOptions MetricsOptions
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
			ProxyOptions:           defaultProxyOptions,
			SaveMatchedRoutePath:   true,
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
			BasePath:               os.Getenv("BASE_PATH"),
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
	bw := a.AppBuiltWith
	if len(bw) > 0 {
		bw = fmt.Sprintf(" (built with %s)", bw)
	}
	return fmt.Sprintf("%s %s%s", a.AppName, a.AppVer, bw)
}

// basePath returns base path of the application
func (a *App) basePath() string {
	a.pathLock.RLock()
	defer a.pathLock.RUnlock()

	if a.originalBasePath == nil || *a.originalBasePath != a.RouterOptions.BasePath {
		a.pathLock.RUnlock()
		a.pathLock.Lock()

		a.originalBasePath = &a.RouterOptions.BasePath
		a.fixedBasePath = a.RouterOptions.BasePath
		// Add leading slash
		if len(a.fixedBasePath) > 0 && a.fixedBasePath[0] != '/' {
			a.fixedBasePath = "/" + a.fixedBasePath
		}
		// Strip trailing slash
		if len(a.fixedBasePath) > 1 && a.fixedBasePath[len(a.fixedBasePath)-1] == '/' {
			a.fixedBasePath = a.fixedBasePath[:len(a.fixedBasePath)-1]
		}

		a.pathLock.Unlock()
		a.pathLock.RLock()
	}
	return a.fixedBasePath
}

// Start web application
func (a *App) Start( /*config *server.Configuration*/ ) error {
	if err := a.initLogger(); err != nil {
		return err
	}

	addr := "0.0.0.0" // config.Address
	if addr == "0.0.0.0" {
		addr = ""
	}

	name := a.AppName
	if len(name) == 0 {
		name = "Azugo"
	}

	a.Log().Info(a.String())

	a.Log().Info(fmt.Sprintf("Listening on %s:%d...", "0.0.0.0", 3000)) // config.Address, config.Port)

	server := &fasthttp.Server{
		NoDefaultServerHeader:        true,
		Handler:                      a.Handler,
		Logger:                       zap.NewStdLog(a.Log().Named("http")),
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
	}

	http2.ConfigureServer(server, http2.ServerConfig{})

	err := server.ListenAndServe(fmt.Sprintf("%s:%d", addr, 3000)) // config.Port))
	if err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}
	return nil
}

// Stop application and its services
func (a *App) Stop() {
	a.bgstop()
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
