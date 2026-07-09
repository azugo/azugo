package middleware

import (
	"io"
	"net"
	"testing"

	"azugo.io/azugo"
	"azugo.io/core/http"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newBenchApp creates an app with the default middleware stack, in the same
// order server.New registers it: RealIP outer, RequestLogger inner.
func newBenchApp(b *testing.B) *azugo.TestApp {
	b.Helper()

	a := azugo.NewTestApp()
	a.StartBenchmark()
	b.Cleanup(a.Stop)

	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	_ = a.App.ReplaceLogger(zap.New(zapcore.NewCore(enc, zapcore.AddSync(io.Discard), zap.InfoLevel)))

	a.UsePriority(RealIP)
	a.UsePriority(RequestLogger)

	return a
}

// newProdBenchApp mirrors what a.ApplyConfig() sets from default configuration
// (server.New + ApplyConfig), unlike newBenchApp/NewTestApp which force
// Proxy.TrustAll and leave TrustedHeaders empty - that skips the
// RealIP/X-Forwarded-For resolution path entirely, so it doesn't reflect a
// real deployment's per-request cost.
func newProdBenchApp(b *testing.B) *azugo.TestApp {
	b.Helper()

	a := newBenchApp(b)

	opts := a.RouterOptions()
	opts.Proxy.TrustAll = false
	opts.Proxy.TrustedIPs = []net.IP{net.IPv4(127, 0, 0, 1)}
	opts.Proxy.TrustedHeaders = []string{http.HeaderRealIP, http.HeaderXForwardedFor}
	opts.Proxy.ForwardLimit = 1

	return a
}

// benchCtxFromLoopback builds a request context whose remote address is
// 127.0.0.1, matching the trusted-proxy IP used above.
func benchCtxFromLoopback(method, uri string) *fasthttp.RequestCtx {
	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod(method)
	ctx.Request.SetRequestURI(uri)
	ctx.SetRemoteAddr(&net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)})

	return ctx
}

func BenchmarkDefaultStack(b *testing.B) {
	a := newBenchApp(b)
	a.Get("/user/{id}", func(ctx *azugo.Context) {
		ctx.JSON(struct {
			ID int `json:"id"`
		}{ID: 1})
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/user/15")

	b.ReportAllocs()

	for b.Loop() {
		a.App.Handler(ctx)
	}
}

// BenchmarkDefaultStackProdProxy mirrors the trusted-proxy config a real
// server.New()+ApplyConfig() deployment uses by default (TrustedHeaders set,
// request over trusted loopback, no X-Forwarded-For/X-Real-IP header
// present) - the actual per-request cost RealIP pays in production.
func BenchmarkDefaultStackProdProxy(b *testing.B) {
	a := newProdBenchApp(b)
	a.Get("/user/{id}", func(ctx *azugo.Context) {
		ctx.JSON(struct {
			ID int `json:"id"`
		}{ID: 1})
	})

	ctx := benchCtxFromLoopback("GET", "/user/15")

	b.ReportAllocs()

	for b.Loop() {
		a.App.Handler(ctx)
	}
}

// BenchmarkRealIPProdProxy isolates RealIP's own per-request cost (no
// RequestLogger noise) under the same production trusted-proxy config as
// BenchmarkDefaultStackProdProxy.
func BenchmarkRealIPProdProxy(b *testing.B) {
	a := newProdBenchApp(b)
	a.Get("/user/{id}", func(ctx *azugo.Context) {
		ctx.SkipRequestLog()
		ctx.JSON(struct {
			ID int `json:"id"`
		}{ID: 1})
	})

	ctx := benchCtxFromLoopback("GET", "/user/15")

	b.ReportAllocs()

	for b.Loop() {
		a.App.Handler(ctx)
	}
}

func BenchmarkDefaultStackSkipLog(b *testing.B) {
	a := newBenchApp(b)
	a.Get("/user/{id}", func(ctx *azugo.Context) {
		ctx.SkipRequestLog()
		ctx.JSON(struct {
			ID int `json:"id"`
		}{ID: 1})
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/user/15")

	b.ReportAllocs()

	for b.Loop() {
		a.App.Handler(ctx)
	}
}
