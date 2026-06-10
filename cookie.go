package azugo

import (
	"iter"
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

// cookieDefaultSecurity is the underlying type for the CookieDefaultSecurity option.
type cookieDefaultSecurity struct{}

func (cookieDefaultSecurity) apply(_ *fasthttp.Cookie) {}

// CookieDefaultSecurity applies secure-by-default attributes to a response cookie
// and automatically selects the strongest cookie-name prefix the request allows:
//
//   - __Host- when the cookie is Secure and the application is served from the
//     root base path: forces Secure, Path="/" and no Domain.
//   - __Secure- when the cookie is Secure but served under a non-root base path:
//     forces Secure and scopes Path to the base path.
//   - no prefix when the cookie is not Secure (development over plain HTTP).
//
// It also sets HttpOnly, an empty Domain (host-only) and SameSite=Lax.
//
// SameSite may be overridden by passing an explicit option. The prefix-mandated
// attributes (Secure, Path, Domain) and HttpOnly are enforced after all other
// options and cannot be downgraded.
func CookieDefaultSecurity() CookieOption {
	return cookieDefaultSecurity{}
}

// CookieCtx provides cookie read and write helpers on an HTTP context.
type CookieCtx struct {
	noCopy noCopy

	ctx *Context
}

// Get the value of the request cookie.
//
// Returns empty value if no cookie is present.
func (c *CookieCtx) Get(name string) string {
	return utils.B2S(c.ctx.Request().Header.Cookie(name))
}

// Keys returns an iterator over all request cookie names.
func (c *CookieCtx) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for k := range c.ctx.Request().Header.Cookies() {
			if !yield(utils.B2S(k)) {
				return
			}
		}
	}
}

// All returns an iterator over all request cookies as name and value pairs.
func (c *CookieCtx) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for k, v := range c.ctx.Request().Header.Cookies() {
			if !yield(utils.B2S(k), utils.B2S(v)) {
				return
			}
		}
	}
}

// writeCookie writes cookie with all options applied.
func (c *CookieCtx) writeCookie(name string, setup func(*fasthttp.Cookie), opts []CookieOption) {
	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)

	defaultSecurity := false

	for _, opt := range opts {
		if _, ok := opt.(cookieDefaultSecurity); ok {
			defaultSecurity = true

			break
		}
	}

	var secure bool

	if defaultSecurity {
		secure = c.ctx.IsTLS() || !c.ctx.Env().IsDevelopment()
		if secure {
			if c.ctx.BasePath() == "" {
				// __Host- requires Secure, no Domain and Path="/".
				name = "__Host-" + name
			} else {
				// __Secure- requires Secure but allows a scoped Path.
				name = "__Secure-" + name
			}
		}
	}

	cookie.SetKey(name)
	setup(cookie)

	if defaultSecurity {
		cookie.SetSameSite(fasthttp.CookieSameSiteLaxMode)
	}

	for _, opt := range opts {
		opt.apply(cookie)
	}

	if defaultSecurity {
		// Enforce secure defaults
		cookie.SetHTTPOnly(true)
		cookie.SetDomain("")
		cookie.SetPath(c.ctx.BasePath())

		if secure {
			cookie.SetSecure(true)
		}
	}

	c.ctx.Response().Header.SetCookie(cookie)
}

// Set a response cookie with the given name and value.
func (c *CookieCtx) Set(name, value string, opts ...CookieOption) {
	c.writeCookie(name, func(cookie *fasthttp.Cookie) {
		cookie.SetValue(value)
	}, opts)
}

// Clear the named cookie on the client.
// Pass the same Path and Domain options that were used when setting the cookie.
func (c *CookieCtx) Clear(name string, opts ...CookieOption) {
	c.writeCookie(name, func(cookie *fasthttp.Cookie) {
		cookie.SetExpire(fasthttp.CookieExpireDelete)
	}, opts)
}
