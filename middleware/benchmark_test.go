package middleware

import (
	"io"
	"testing"

	"azugo.io/azugo"

	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// newBenchApp creates an app with the default middleware stack and a real JSON
// encoder writing to io.Discard so that log encoding cost is measured.
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
