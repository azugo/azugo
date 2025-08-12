package azugo

import (
	"embed"
	"strings"
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

//go:embed testdata/*
var testdata embed.FS

func TestRouterStatic(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	// Test other method than GET (want 308 instead of 301)
	a.StaticEmbedded("/", testdata, StaticDirTrimPrefix("testdata/"), StaticContentReplacer(func(ctx *Context) (string, *strings.Replacer) {
		return "cached-", strings.NewReplacer("{{BASE_URL}}", ctx.BaseURL(), "{{BASE_PATH}}", ctx.BasePath())
	}))

	resp, err := a.TestClient().Call(fasthttp.MethodGet, "/index.html", nil)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
	qt.Check(t, qt.Equals(string(resp.Header.ContentType()), "text/html; charset=utf-8"))
	qt.Check(t, qt.StringContains(string(resp.Body()), `var baseURL = "http://test";`))
	qt.Check(t, qt.StringContains(string(resp.Body()), `var basePath = "";`))
}
