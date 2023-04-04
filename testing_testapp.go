package azugo

import (
	"net"
	"testing"

	"azugo.io/azugo/config"
	"azugo.io/core"

	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

// TestApp represents testing app instance
type TestApp struct {
	*App

	ln   *fasthttputil.InmemoryListener
	logs *observer.ObservedLogs
}

// NewTestApp creates new testing application instance
func NewTestApp(app ...*App) *TestApp {
	var a *App
	if len(app) == 0 {
		a = New()
		a.AppName = "Azugo TestApp"
		a.AppVer = "1.0"
	} else {
		a = app[0]
	}

	// Trust all proxy headers for test app
	a.defaultMux.RouterOptions.Proxy.TrustAll = true

	conf := config.New()
	a.SetConfig(nil, conf)
	a.App.SetConfig(nil, conf.Core())
	_ = conf.Load(nil, conf, string(core.EnvironmentDevelopment))

	return &TestApp{
		App: a,
	}
}

func (a *TestApp) initLogs() {
	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	_ = a.ReplaceLogger(zap.New(observedZapCore))

	a.logs = observedLogs
}

func (a *TestApp) applyConfig() {
	if len(a.Config().Server.Path) > 0 {
		a.RouterOptions().BasePath = a.Config().Server.Path
	}
}

// Start starts testing web server instance
func (a *TestApp) Start(t *testing.T) {
	a.applyConfig()
	a.initLogs()
	require.NoError(t, a.App.App.Start())

	server := &fasthttp.Server{
		NoDefaultServerHeader:        true,
		Handler:                      a.App.Handler,
		Logger:                       zap.NewStdLog(a.App.Log().Named("http")),
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
	}
	ln := fasthttputil.NewInmemoryListener()
	go func() {
		require.NoError(t, server.Serve(ln))
	}()
	a.ln = ln
}

// StartBenchmark starts benchmarking web server instance
func (a *TestApp) StartBenchmark() {
	a.applyConfig()
	a.initLogs()
	if err := a.App.App.Start(); err != nil {
		panic(err)
	}

	server := &fasthttp.Server{
		NoDefaultServerHeader:        true,
		Handler:                      a.App.Handler,
		Logger:                       zap.NewStdLog(a.App.Log().Named("http")),
		StreamRequestBody:            true,
		DisablePreParseMultipartForm: true,
	}
	ln := fasthttputil.NewInmemoryListener()
	go func() {
		if err := server.Serve(ln); err != nil {
			panic(err)
		}
	}()
	a.ln = ln
}

// Stop web server instance
func (a *TestApp) Stop() {
	if a.ln != nil {
		a.ln.Close()
	}
	a.App.Stop()
}

// TestClient returns testing client that will do HTTP requests to test web server
func (a *TestApp) TestClient() *TestClient {
	client := &fasthttp.Client{}
	client.Dial = func(addr string) (net.Conn, error) {
		return a.ln.Dial()
	}

	return &TestClient{
		app:    a,
		client: client,
	}
}
