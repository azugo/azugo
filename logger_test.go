package azugo

import (
	"testing"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"go.uber.org/zap"
)

func TestContextLogBaseFields(t *testing.T) {
	app := NewTestApp()
	app.Start(t)
	defer app.Stop()

	app.Get("/test", func(ctx *Context) {
		ctx.Log().Info("test message")
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := app.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))

	entries := app.logs.FilterMessage("test message").All()
	qt.Assert(t, qt.HasLen(entries, 1))

	fields := entries[0].ContextMap()
	qt.Check(t, qt.Equals(fields["http.request.method"], any("GET")))
	qt.Check(t, qt.Equals(fields["url.path"], any("/test")))

	id, ok := fields["http.request.id"].(string)
	qt.Assert(t, qt.IsTrue(ok))
	qt.Check(t, qt.HasLen(id, 26))

	ip, ok := fields["source.ip"].(string)
	qt.Assert(t, qt.IsTrue(ok))
	qt.Check(t, qt.Not(qt.Equals(ip, "")))
}

func TestContextLogAddedFieldsOverrideBase(t *testing.T) {
	app := NewTestApp()
	app.Start(t)
	defer app.Stop()

	app.Get("/test", func(ctx *Context) {
		_ = ctx.AddLogFields(
			zap.String("source.ip", "10.1.2.3"),
			zap.String("user.id", "tester"),
		)

		ctx.Log().Info("test message")
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := app.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))

	entries := app.logs.FilterMessage("test message").All()
	qt.Assert(t, qt.HasLen(entries, 1))

	fields := entries[0].ContextMap()
	qt.Check(t, qt.Equals(fields["source.ip"], any("10.1.2.3")))
	qt.Check(t, qt.Equals(fields["user.id"], any("tester")))
	qt.Check(t, qt.Equals(fields["http.request.method"], any("GET")))
}
