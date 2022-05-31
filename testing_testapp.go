package azugo

import (
	"net"
	"testing"

	"azugo.io/azugo/config"

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
	} else {
		a = app[0]
	}

	// Trust all proxy headers for test app
	a.RouterOptions.ProxyOptions.TrustAll = true

	observedZapCore, observedLogs := observer.New(zap.InfoLevel)
	a.logger = zap.New(observedZapCore)

	conf := config.New()
	a.SetConfig(nil, conf)
	_ = conf.Load(conf, string(EnvironmentDevelopment))

	return &TestApp{
		App:  a,
		logs: observedLogs,
	}
}

// Start starts testing web server instance
func (a *TestApp) Start(t *testing.T) {
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
	if err := a.App.initLogger(); err != nil {
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
