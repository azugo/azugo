package proxy

import (
	"github.com/valyala/fasthttp"
)

// RewriteCookies rewrites cookies in the response.
func RewriteCookies(tls bool, host string, resp *fasthttp.Response) {
	resp.Header.VisitAllCookie(func(k, v []byte) {
		cookie := fasthttp.AcquireCookie()
		defer fasthttp.ReleaseCookie(cookie)

		cookie.SetKeyBytes(k)

		if err := cookie.ParseBytes(v); err != nil {
			return
		}
		// Downgrade cookie to not secure
		if !tls && cookie.Secure() {
			cookie.SetSecure(false)
		}
		// Change cookie domain
		if cookie.Domain() != nil {
			cookie.SetDomain(host)
		}

		resp.Header.SetCookie(cookie)
	})
}
