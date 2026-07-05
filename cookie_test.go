package azugo

import (
	"testing"

	"azugo.io/core"
	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func parseCookie(t *testing.T, raw []byte) *fasthttp.Cookie {
	t.Helper()

	c := fasthttp.AcquireCookie()
	t.Cleanup(func() { fasthttp.ReleaseCookie(c) })
	qt.Assert(t, qt.IsNil(c.ParseBytes(raw)))

	return c
}

func TestCookieGet(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		qt.Check(t, qt.Equals(ctx.Cookie.Get("session"), "abc123"))
		ctx.StatusCode(http.StatusNoContent)
	})

	resp, err := a.TestClient().Get("/", a.TestClient().WithCookie("session", "abc123"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusNoContent))
}

func TestCookieKeysAndAll(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		names := make([]string, 0, 2)
		for name := range ctx.Cookie.Keys() {
			names = append(names, name)
		}
		qt.Check(t, qt.ContentEquals(names, []string{"session", "theme"}))

		all := make(map[string]string)
		for name, value := range ctx.Cookie.All() {
			all[name] = value
		}
		qt.Check(t, qt.DeepEquals(all, map[string]string{"session": "abc123", "theme": "dark"}))

		ctx.StatusCode(http.StatusNoContent)
	})

	tc := a.TestClient()

	resp, err := tc.Get("/", tc.WithCookie("session", "abc123"), tc.WithCookie("theme", "dark"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusNoContent))
}

func TestCookieSet(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123", CookieHTTPOnly(true), CookieSecure(true), CookieSameSiteLax)
	})

	resp, err := a.TestClient().Get("/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(c.Key()), "session"))
	qt.Check(t, qt.Equals(string(c.Value()), "abc123"))
	qt.Check(t, qt.IsTrue(c.HTTPOnly()))
	qt.Check(t, qt.IsTrue(c.Secure()))
	qt.Check(t, qt.Equals(c.SameSite(), fasthttp.CookieSameSiteLaxMode))
}

func TestCookieClear(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Clear("session", CookiePath("/"))
	})

	resp, err := a.TestClient().Get("/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(c.Key()), "session"))
	qt.Check(t, qt.Equals(string(c.Path()), "/"))
	qt.Check(t, qt.IsTrue(c.Expire().Before(fasthttp.CookieExpireDelete.Add(1))))
}

func TestCookieDefaultSecurityHostPrefix(t *testing.T) {
	// Default test app runs in Production from the root base path, so the
	// strongest __Host- prefix is selected.
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123", CookieDefaultSecurity())
	})

	resp, err := a.TestClient().Get("/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(c.Key()), "__Host-session"))
	qt.Check(t, qt.IsTrue(c.HTTPOnly()))
	qt.Check(t, qt.IsTrue(c.Secure()))
	qt.Check(t, qt.Equals(string(c.Domain()), ""))
	qt.Check(t, qt.Equals(string(c.Path()), "/"))
	qt.Check(t, qt.Equals(c.SameSite(), fasthttp.CookieSameSiteLaxMode))
}

func TestCookieDefaultSecuritySecurePrefix(t *testing.T) {
	// A non-root base path cannot satisfy __Host- (Path must be "/"), so the
	// __Secure- prefix is selected with the base path as scope.
	a := NewTestApp()
	a.Config().Server.Path = "/test"
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123", CookieDefaultSecurity())
	})

	resp, err := a.TestClient().Get("/test/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(c.Key()), "__Secure-session"))
	qt.Check(t, qt.IsTrue(c.Secure()))
	qt.Check(t, qt.Equals(string(c.Domain()), ""))
	qt.Check(t, qt.Equals(string(c.Path()), "/test"))
}

func TestCookieDefaultSecurityDevelopment(t *testing.T) {
	// Environment is read at app construction, so set it before NewTestApp.
	t.Setenv("ENVIRONMENT", string(core.EnvironmentDevelopment))

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123", CookieDefaultSecurity())
	})

	tc := a.TestClient()

	// Plain HTTP in development: no Secure flag and no prefix.
	resp, err := tc.Get("/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.IsFalse(c.Secure()))
	qt.Check(t, qt.Equals(string(c.Key()), "session"))
	qt.Check(t, qt.IsTrue(c.HTTPOnly()))

	// A TLS request (proxy-forwarded) sets Secure and the __Host- prefix even in development.
	respTLS, err := tc.Get("/", tc.WithHeader(http.HeaderXForwardedProto, "https"))
	defer fasthttp.ReleaseResponse(respTLS)
	qt.Assert(t, qt.IsNil(err))
	cTLS := parseCookie(t, respTLS.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.IsTrue(cTLS.Secure()))
	qt.Check(t, qt.Equals(string(cTLS.Key()), "__Host-session"))
}

func TestCookieDefaultSecurityNoDowngrade(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		// SameSite may be tightened, but the prefix-mandated attributes must not
		// be downgraded by explicit options.
		ctx.Cookie.Set("session", "abc123",
			CookieDefaultSecurity(),
			CookieSameSiteStrict,
			CookieSecure(false),
			CookiePath("/custom"),
			CookieDomain("example.com"),
		)
	})

	resp, err := a.TestClient().Get("/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	// Tightening is honored.
	qt.Check(t, qt.Equals(c.SameSite(), fasthttp.CookieSameSiteStrictMode))
	// Downgrades are ignored: __Host- contract is preserved.
	qt.Check(t, qt.Equals(string(c.Key()), "__Host-session"))
	qt.Check(t, qt.IsTrue(c.Secure()))
	qt.Check(t, qt.Equals(string(c.Path()), "/"))
	qt.Check(t, qt.Equals(string(c.Domain()), ""))
}

func TestCookieJar(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/set", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123")
	})
	a.Get("/get", func(ctx *Context) {
		qt.Check(t, qt.Equals(ctx.Cookie.Get("session"), "abc123"))
		ctx.StatusCode(http.StatusNoContent)
	})

	c := a.TestClient()

	resp, err := c.Get("/set")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(c.Cookies()["session"], "abc123"))

	resp2, err := c.Get("/get")
	defer fasthttp.ReleaseResponse(resp2)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp2.StatusCode(), http.StatusNoContent))
}

func TestCookieJarClear(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/set", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123")
	})
	a.Get("/clear", func(ctx *Context) {
		ctx.Cookie.Clear("session")
	})

	c := a.TestClient()

	resp, err := c.Get("/set")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(c.Cookies()["session"], "abc123"))

	resp2, err := c.Get("/clear")
	defer fasthttp.ReleaseResponse(resp2)
	qt.Assert(t, qt.IsNil(err))
	_, hasSession := c.Cookies()["session"]
	qt.Check(t, qt.IsFalse(hasSession))
}
