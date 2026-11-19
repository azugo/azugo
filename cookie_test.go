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

	// A secure request carries the cookie under the __Host- name Set writes.
	resp, err := a.TestClient().Get("/", a.TestClient().WithCookie("__Host-session", "abc123"))
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

	resp, err := tc.Get("/", tc.WithCookie("__Host-session", "abc123"), tc.WithCookie("__Host-theme", "dark"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusNoContent))
}

func TestCookieSet(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123", CookieInsecure(), CookieHTTPOnly(true), CookieSecure(true), CookieSameSiteLax)
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
		ctx.Cookie.Clear("session", CookieInsecure(), CookiePath("/"))
	})

	resp, err := a.TestClient().Get("/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(c.Key()), "session"))
	qt.Check(t, qt.Equals(string(c.Path()), "/"))
	qt.Check(t, qt.IsTrue(c.Expire().Before(fasthttp.CookieExpireDelete.Add(1))))
}

func TestCookieSecureHostPrefix(t *testing.T) {
	// Default test app runs in Production from the root base path, so the
	// strongest __Host- prefix is selected.
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123")
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

func TestCookieSecureSecurePrefix(t *testing.T) {
	// A non-root base path cannot satisfy __Host- (Path must be "/"), so the
	// __Secure- prefix is selected with the base path as scope.
	a := NewTestApp()
	a.Config().Server.Path = "/test"
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123")
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

func TestCookieSecureDevelopment(t *testing.T) {
	// Environment is read at app construction, so set it before NewTestApp.
	t.Setenv("ENVIRONMENT", string(core.EnvironmentDevelopment))

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123")
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

func TestCookieSecureExplicitPath(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		// A scoped Path cannot satisfy __Host-, so the __Secure- prefix keeps the Path.
		ctx.Cookie.Set("session", "abc123", CookiePath("/auth"))
	})

	a.Get("/root", func(ctx *Context) {
		ctx.Cookie.Set("session", "abc123", CookiePath("/"))
	})

	tc := a.TestClient()

	resp, err := tc.Get("/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(c.Key()), "__Secure-session"))
	qt.Check(t, qt.IsTrue(c.Secure()))
	qt.Check(t, qt.IsTrue(c.HTTPOnly()))
	qt.Check(t, qt.Equals(string(c.Domain()), ""))
	qt.Check(t, qt.Equals(string(c.Path()), "/auth"))

	respRoot, err := tc.Get("/root")
	defer fasthttp.ReleaseResponse(respRoot)
	qt.Assert(t, qt.IsNil(err))

	cRoot := parseCookie(t, respRoot.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(cRoot.Key()), "__Host-session"))
	qt.Check(t, qt.Equals(string(cRoot.Path()), "/"))
}

func TestCookieSecureNoDowngrade(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		// SameSite may be tightened, but the prefix-mandated attributes must not
		// be downgraded by explicit options.
		ctx.Cookie.Set("session", "abc123",
			CookieSameSiteStrict,
			CookieSecure(false),
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
	qt.Check(t, qt.Equals(c.Cookies()["__Host-session"], "abc123"))

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
	qt.Check(t, qt.Equals(c.Cookies()["__Host-session"], "abc123"))

	resp2, err := c.Get("/clear")
	defer fasthttp.ReleaseResponse(resp2)
	qt.Assert(t, qt.IsNil(err))
	_, hasSession := c.Cookies()["__Host-session"]
	qt.Check(t, qt.IsFalse(hasSession))
}

func TestCookieGetReadsExactName(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		// Only the name Set writes for this request is read.
		qt.Check(t, qt.Equals(ctx.Cookie.Get("session"), "host"))
		qt.Check(t, qt.Equals(ctx.Cookie.Get("scoped", CookiePath("/auth")), "scoped"))
		// A planted variant under another prefix, or none, is not read in its place.
		qt.Check(t, qt.Equals(ctx.Cookie.Get("theme"), ""))
		qt.Check(t, qt.Equals(ctx.Cookie.Get("plain"), ""))
		qt.Check(t, qt.Equals(ctx.Cookie.Get("other", CookiePath("/auth")), ""))
		// An explicitly prefixed or insecure name is read as given.
		qt.Check(t, qt.Equals(ctx.Cookie.Get("__Host-session"), "host"))
		qt.Check(t, qt.Equals(ctx.Cookie.Get("__Secure-session"), ""))
		qt.Check(t, qt.Equals(ctx.Cookie.Get("plain", CookieInsecure()), "bare"))
		ctx.StatusCode(http.StatusNoContent)
	})

	tc := a.TestClient()

	resp, err := tc.Get("/",
		tc.WithCookie("session", "bare"), tc.WithCookie("__Host-session", "host"),
		tc.WithCookie("__Secure-theme", "secure"), tc.WithCookie("plain", "bare"),
		tc.WithCookie("__Secure-scoped", "scoped"), tc.WithCookie("__Host-other", "host"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusNoContent))
}

// clearedCookies parses every Set-Cookie header of resp into a map keyed by cookie name.
func clearedCookies(t *testing.T, resp *fasthttp.Response) map[string]*fasthttp.Cookie {
	t.Helper()

	out := make(map[string]*fasthttp.Cookie)

	for key, value := range resp.Header.Cookies() {
		out[string(key)] = parseCookie(t, value)
	}

	return out
}

func TestCookieClearDefaultSecurityClearsPresentVariants(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Cookie.Clear("session")
	})

	tc := a.TestClient()

	// Both a __Host- cookie (set over TLS) and a bare one (set in development) are removed.
	resp, err := tc.Get("/", tc.WithCookie("__Host-session", "x"), tc.WithCookie("session", "y"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	cleared := clearedCookies(t, resp)
	qt.Assert(t, qt.HasLen(cleared, 2))

	host := cleared["__Host-session"]
	qt.Assert(t, qt.IsNotNil(host))
	qt.Check(t, qt.Equals(string(host.Path()), "/"))
	qt.Check(t, qt.IsTrue(host.Secure()))
	qt.Check(t, qt.IsTrue(host.Expire().Before(fasthttp.CookieExpireDelete.Add(1))))

	bare := cleared["session"]
	qt.Assert(t, qt.IsNotNil(bare))
	qt.Check(t, qt.IsTrue(bare.Expire().Before(fasthttp.CookieExpireDelete.Add(1))))

	// Without any variant in the request the name the request's security would produce is cleared.
	resp2, err := tc.Get("/")
	defer fasthttp.ReleaseResponse(resp2)
	qt.Assert(t, qt.IsNil(err))

	cleared = clearedCookies(t, resp2)
	qt.Assert(t, qt.HasLen(cleared, 1))
	qt.Check(t, qt.IsNotNil(cleared["__Host-session"]))
}

func TestCookieKeysAndAllMatchGet(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		names := make([]string, 0, 1)
		for name := range ctx.Cookie.Keys() {
			names = append(names, name)
		}
		// Only what Get reads, under its bare name; planted variants are skipped.
		qt.Check(t, qt.DeepEquals(names, []string{"session"}))

		all := make(map[string]string)
		for name, value := range ctx.Cookie.All() {
			all[name] = value
			qt.Check(t, qt.Equals(ctx.Cookie.Get(name), value))
		}
		qt.Check(t, qt.DeepEquals(all, map[string]string{"session": "host"}))

		ctx.StatusCode(http.StatusNoContent)
	})

	tc := a.TestClient()

	resp, err := tc.Get("/",
		tc.WithCookie("session", "bare"), tc.WithCookie("__Host-session", "host"), tc.WithCookie("__Secure-theme", "dark"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusNoContent))
}

func TestCookieSetKeepsExplicitPrefix(t *testing.T) {
	a := NewTestApp()
	a.Config().Server.Path = "/test"
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		// An already-prefixed name is not prefixed twice and keeps the attributes its prefix mandates.
		ctx.Cookie.Set("__Host-session", "abc")
	})

	resp, err := a.TestClient().Get("/test/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(c.Key()), "__Host-session"))
	qt.Check(t, qt.Equals(string(c.Path()), "/"))
	qt.Check(t, qt.IsTrue(c.Secure()))
}
