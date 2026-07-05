package azugo

import (
	"embed"
	"strings"
	"testing"

	"azugo.io/core/http"
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
	err := a.StaticEmbedded("/", &testdata, StaticDirTrimPrefix("testdata/"), StaticContentReplacer(func(ctx *Context) (string, *strings.Replacer) {
		return "cached-", strings.NewReplacer("{{BASE_URL}}", ctx.BaseURL(), "{{BASE_PATH}}", ctx.BasePath())
	}))
	qt.Assert(t, qt.IsNil(err))

	resp, err := a.TestClient().Call(http.MethodGet, "/index.html", nil)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
	qt.Check(t, qt.Equals(string(resp.Header.ContentType()), "text/html; charset=utf-8"))
	qt.Check(t, qt.StringContains(string(resp.Body()), `var baseURL = "http://test";`))
	qt.Check(t, qt.StringContains(string(resp.Body()), `var basePath = "";`))
}

func TestRouterStaticSPARouter(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	// Test other method than GET (want 308 instead of 301)
	err := a.StaticEmbedded("/", &testdata, StaticDirTrimPrefix("testdata/"), StaticSPARouterPath("index.html"), StaticContentReplacer(func(ctx *Context) (string, *strings.Replacer) {
		return "cached-", strings.NewReplacer("{{BASE_URL}}", ctx.BaseURL(), "{{BASE_PATH}}", ctx.BasePath())
	}))
	qt.Assert(t, qt.IsNil(err))

	resp, err := a.TestClient().Call(http.MethodGet, "/", nil)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusOK))
	qt.Check(t, qt.Equals(string(resp.Header.ContentType()), "text/html; charset=utf-8"))
	qt.Check(t, qt.StringContains(string(resp.Body()), `var baseURL = "http://test";`))
	qt.Check(t, qt.StringContains(string(resp.Body()), `var basePath = "";`))
}

func TestRouterStaticSPARouterInvalidPath(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	// Test other method than GET (want 308 instead of 301)
	err := a.StaticEmbedded("/", &testdata, StaticDirTrimPrefix("testdata/"), StaticSPARouterPath("index.htm"))
	qt.Assert(t, qt.ErrorMatches(err, "static SPA route handler file not found: .*"))
}
