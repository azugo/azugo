package azugo

import (
	"iter"
	"strings"
	"time"

	"azugo.io/azugo/internal/utils"

	"github.com/valyala/fasthttp"
)

// CookieOption is an option for setting a response cookie.
type CookieOption interface {
	apply(c *fasthttp.Cookie)
}

// CookieSameSite controls the SameSite attribute of a cookie.
type CookieSameSite fasthttp.CookieSameSite

const (
	// CookieSameSiteDisabled omits the SameSite attribute.
	CookieSameSiteDisabled CookieSameSite = CookieSameSite(fasthttp.CookieSameSiteDisabled)
	// CookieSameSiteDefault sets SameSite without a mode qualifier.
	CookieSameSiteDefault CookieSameSite = CookieSameSite(fasthttp.CookieSameSiteDefaultMode)
	// CookieSameSiteLax sets SameSite=Lax.
	CookieSameSiteLax CookieSameSite = CookieSameSite(fasthttp.CookieSameSiteLaxMode)
	// CookieSameSiteStrict sets SameSite=Strict.
	CookieSameSiteStrict CookieSameSite = CookieSameSite(fasthttp.CookieSameSiteStrictMode)
	// CookieSameSiteNone sets SameSite=None (requires Secure=true).
	CookieSameSiteNone CookieSameSite = CookieSameSite(fasthttp.CookieSameSiteNoneMode)
)

func (o CookieSameSite) apply(c *fasthttp.Cookie) {
	c.SetSameSite(fasthttp.CookieSameSite(o))
}

// CookiePath sets the Path attribute of a response cookie.
type CookiePath string

func (o CookiePath) apply(c *fasthttp.Cookie) {
	c.SetPath(string(o))
}

// CookieDomain sets the Domain attribute of a response cookie.
type CookieDomain string

func (o CookieDomain) apply(c *fasthttp.Cookie) {
	c.SetDomain(string(o))
}

// CookieMaxAge sets the Max-Age attribute of a response cookie.
type CookieMaxAge time.Duration

func (o CookieMaxAge) apply(c *fasthttp.Cookie) {
	c.SetMaxAge(int(time.Duration(o).Seconds()))
}

// CookieExpires sets the Expires attribute of a response cookie.
type CookieExpires time.Time

func (o CookieExpires) apply(c *fasthttp.Cookie) {
	c.SetExpire(time.Time(o))
}

// CookieHTTPOnly sets or clears the HttpOnly flag on a response cookie.
type CookieHTTPOnly bool

func (o CookieHTTPOnly) apply(c *fasthttp.Cookie) {
	c.SetHTTPOnly(bool(o))
}

// CookieSecure sets or clears the Secure flag on a response cookie.
type CookieSecure bool

func (o CookieSecure) apply(c *fasthttp.Cookie) {
	c.SetSecure(bool(o))
}

// cookieInsecure is the underlying type for the CookieInsecure option.
type cookieInsecure struct{}

func (cookieInsecure) apply(_ *fasthttp.Cookie) {}

// CookieInsecure opts a cookie out of the default security CookieCtx applies: the name is used as
// given and only the options passed set its attributes. Pass it to Get and Clear as well.
func CookieInsecure() CookieOption {
	return cookieInsecure{}
}

// Cookie-name prefixes (RFC 6265bis section 4.1.3) applied by default.
const (
	cookiePrefixHost   = "__Host-"
	cookiePrefixSecure = "__Secure-"
)

func hasCookiePrefix(name string) bool {
	return strings.HasPrefix(name, cookiePrefixHost) || strings.HasPrefix(name, cookiePrefixSecure)
}

func bareCookieName(name string) string {
	return strings.TrimPrefix(strings.TrimPrefix(name, cookiePrefixHost), cookiePrefixSecure)
}

func isInsecure(opts []CookieOption) bool {
	for _, opt := range opts {
		if _, ok := opt.(cookieInsecure); ok {
			return true
		}
	}

	return false
}

// CookieCtx provides cookie read and write helpers on an HTTP context.
//
// Unless CookieInsecure is passed, a cookie is secure by default and its name gets the strongest
// prefix the request allows:
//
//   - __Host- when the request is TLS (or the app is not in development) and the cookie is scoped
//     to the root path: forces Secure, Path="/" and no Domain.
//   - __Secure- when the request is TLS (or the app is not in development) and the cookie is
//     scoped below the root, by a non-root base path or a CookiePath: forces Secure.
//   - no prefix in development over plain HTTP.
//
// It also sets HttpOnly, an empty Domain (host-only) and SameSite=Lax. SameSite and Path may be
// set by passing an option; Secure, Domain and HttpOnly cannot be downgraded. Get and Clear read
// and clear exactly the name Set writes, so pass them the same CookiePath.
type CookieCtx struct {
	noCopy noCopy

	ctx *Context
}

// Get the value of the request cookie named as Set would name it with the same options; another
// prefix variant, or the bare name, is never read in its place.
//
// Returns empty value if no cookie is present.
func (c *CookieCtx) Get(name string, opts ...CookieOption) string {
	name, _, _ = c.cookieName(name, opts)

	return utils.B2S(c.ctx.Request().Header.Cookie(name))
}

// Keys returns an iterator over the bare names of the request cookies Get reads without options.
func (c *CookieCtx) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for name := range c.All() {
			if !yield(name) {
				return
			}
		}
	}
}

// All returns an iterator over the request cookies Get reads without options, as bare name and
// value pairs; a variant under another prefix, or none, is skipped as Get skips it.
func (c *CookieCtx) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for k, v := range c.ctx.Request().Header.Cookies() {
			key := utils.B2S(k)
			name := bareCookieName(key)

			if written, _, _ := c.cookieName(name, nil); written != key {
				continue
			}

			if !yield(name, utils.B2S(v)) {
				return
			}
		}
	}
}

// cookieName returns the name, path and Secure flag a cookie is written with: unless opts hold
// CookieInsecure, the name gets the prefix the request and path allow, if it has none yet.
func (c *CookieCtx) cookieName(name string, opts []CookieOption) (string, string, bool) {
	if isInsecure(opts) {
		return name, "", false
	}

	path := c.ctx.BasePath()

	for _, opt := range opts {
		if p, ok := opt.(CookiePath); ok {
			path = string(p)
		}
	}

	secure := c.ctx.IsTLS() || !c.ctx.Env().IsDevelopment()

	switch {
	case !secure || hasCookiePrefix(name):
		return name, path, secure
	case path == "" || path == "/":
		// __Host- requires Secure, no Domain and Path="/".
		return cookiePrefixHost + name, path, secure
	default:
		// __Secure- requires Secure but allows a scoped Path.
		return cookiePrefixSecure + name, path, secure
	}
}

// writeCookie writes cookie with all options applied. Unless keepName is set the name is the one
// cookieName returns; a secured cookie always gets the attributes its prefix mandates.
func (c *CookieCtx) writeCookie(name string, keepName bool, setup func(*fasthttp.Cookie), opts []CookieOption) {
	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)

	secured := !isInsecure(opts)

	prefixed, path, secure := c.cookieName(name, opts)
	if !keepName {
		name = prefixed
	}

	cookie.SetKey(name)
	setup(cookie)

	if secured {
		cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	}

	for _, opt := range opts {
		opt.apply(cookie)
	}

	if secured {
		// Enforce secure defaults
		cookie.SetHTTPOnly(true)
		cookie.SetDomain("")
		cookie.SetPath(path)

		if strings.HasPrefix(name, cookiePrefixHost) {
			cookie.SetPath("/")
		}

		if secure || hasCookiePrefix(name) {
			cookie.SetSecure(true)
		}
	}

	c.ctx.Response().Header.SetCookie(cookie)
}

// Set a response cookie with the given name and value.
func (c *CookieCtx) Set(name, value string, opts ...CookieOption) {
	c.writeCookie(name, false, func(cookie *fasthttp.Cookie) {
		cookie.SetValue(value)
	}, opts)
}

// Clear the named cookie on the client. Pass the same options that were used when setting the
// cookie. A secure cookie is cleared under every prefix variant the request carries, so one set
// under a different scheme (e.g. __Host- over TLS) is removed as well.
func (c *CookieCtx) Clear(name string, opts ...CookieOption) {
	expire := func(cookie *fasthttp.Cookie) {
		cookie.SetExpire(fasthttp.CookieExpireDelete)
	}

	if isInsecure(opts) || hasCookiePrefix(name) {
		c.writeCookie(name, false, expire, opts)

		return
	}

	cleared := false

	for k := range c.ctx.Request().Header.Cookies() {
		if ks := utils.B2S(k); bareCookieName(ks) == name {
			c.writeCookie(ks, true, expire, opts)

			cleared = true
		}
	}

	if !cleared {
		c.writeCookie(name, false, expire, opts)
	}
}
