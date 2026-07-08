package middleware

import (
	"testing"

	"azugo.io/azugo"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func newObservedApp(t *testing.T) (*azugo.TestApp, *observer.ObservedLogs) {
	t.Helper()

	a := azugo.NewTestApp()
	a.Start(t)
	t.Cleanup(a.Stop)

	observedCore, observedLogs := observer.New(zap.InfoLevel)
	_ = a.ReplaceLogger(zap.New(observedCore))

	a.UsePriority(RealIP)
	a.UsePriority(RequestLogger)

	return a, observedLogs
}

func TestRequestLoggerMergesContextFields(t *testing.T) {
	a, logs := newObservedApp(t)

	a.Get("/test", func(ctx *azugo.Context) {
		_ = ctx.AddLogFields(zap.String("trace.id", "abc123"))
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))

	entries := logs.FilterMessageSnippet("GET /test").All()
	qt.Assert(t, qt.HasLen(entries, 1))

	fields := entries[0].ContextMap()
	qt.Check(t, qt.Equals(fields["trace.id"], any("abc123")))
	qt.Check(t, qt.Equals(fields["http.request.method"], any("GET")))
}

func TestRequestLoggerDedupesContextFields(t *testing.T) {
	a, logs := newObservedApp(t)

	a.Get("/test", func(ctx *azugo.Context) {
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))

	entries := logs.FilterMessageSnippet("GET /test").All()
	qt.Assert(t, qt.HasLen(entries, 1))

	// RealIP adds "source.ip" via AddLogFields, and RequestLogger also sets its
	// own "source.ip" field directly - the merge must not emit it twice.
	count := 0

	for _, f := range entries[0].Context {
		if f.Key == "source.ip" {
			count++
		}
	}

	qt.Check(t, qt.Equals(count, 1))
}
