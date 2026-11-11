package proxy

import (
	"testing"

	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestRewriteCookiesSkipsInvalidCookie(t *testing.T) {
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	resp.Header.Add(fasthttp.HeaderSetCookie, "bad=1; Max-Age=abc; Domain=upstream.example")
	resp.Header.Add(fasthttp.HeaderSetCookie, "good=2; Domain=upstream.example; Secure")

	RewriteCookies(false, "app.example", resp)

	cookie := fasthttp.AcquireCookie()
	defer fasthttp.ReleaseCookie(cookie)

	cookie.SetKey("good")
	qt.Assert(t, qt.IsTrue(resp.Header.Cookie(cookie)))
	qt.Check(t, qt.Equals(string(cookie.Domain()), "app.example"))
	qt.Check(t, qt.IsFalse(cookie.Secure()))
}
