package azugo

import (
	"testing"
	"time"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

type loginForm struct {
	Username string `json:"username"`
	ReturnTo string `json:"return_to"`
}

// flashApp mounts a POST that flashes and a GET that reads, returning the app.
func flashApp(t *testing.T) *TestApp {
	t.Helper()

	a := NewTestApp()
	a.Start(t)
	t.Cleanup(a.Stop)

	a.Post("/login", func(ctx *Context) {
		ctx.Flash.Error("Sign-in failed.")
		ctx.Flash.Info("Try again.")
		ctx.Flash.Info("Or reset your password.")
		ctx.Flash.FieldError("password", "Invalid username or password.")
		qt.Assert(t, qt.IsNil(ctx.Flash.Set("form", loginForm{Username: "alice", ReturnTo: "/home"})))
		ctx.Redirect("/login")
	})

	a.Get("/login", func(ctx *Context) {
		var form loginForm

		ok, err := ctx.Flash.Get("form", &form)
		qt.Assert(t, qt.IsNil(err))

		if !ctx.Flash.Has() {
			ctx.Text("empty")

			return
		}

		qt.Check(t, qt.IsTrue(ok))
		qt.Check(t, qt.Equals(form.Username, "alice"))
		qt.Check(t, qt.DeepEquals(ctx.Flash.Errors(), []string{"Sign-in failed."}))
		qt.Check(t, qt.DeepEquals(ctx.Flash.Infos(), []string{"Try again.", "Or reset your password."}))
		qt.Check(t, qt.HasLen(ctx.Flash.Warnings(), 0))
		qt.Check(t, qt.Equals(ctx.Flash.FieldErrorFor("password"), "Invalid username or password."))
		qt.Check(t, qt.Equals(ctx.Flash.FieldErrorFor("username"), ""))
		qt.Check(t, qt.DeepEquals(ctx.Flash.FieldErrors(), map[string]string{"password": "Invalid username or password."}))
		ctx.Text("flashed")
	})

	return a
}

func body(t *testing.T, resp *fasthttp.Response) string {
	t.Helper()

	b, err := resp.BodyUncompressed()
	qt.Assert(t, qt.IsNil(err))

	return string(b)
}

func TestFlashPostRedirectGet(t *testing.T) {
	a := flashApp(t)
	tc := a.TestClient()

	resp, err := tc.PostForm("/login", map[string]any{"username": "alice"})
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusSeeOther))
	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "/login"))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	fasthttp.ReleaseResponse(resp)
	// The production test app counts as secure, so the __Host- prefix applies.
	qt.Check(t, qt.Equals(string(c.Key()), "__Host-flash"))
	qt.Check(t, qt.IsTrue(c.HTTPOnly()))
	qt.Check(t, qt.IsTrue(len(c.Value()) >= 22))
	qt.Check(t, qt.IsTrue(c.MaxAge() > 0))

	// The jar carries the cookie; the GET consumes the record and clears the cookie.
	resp2, err := tc.Get("/login")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(body(t, resp2), "flashed"))

	cleared := parseCookie(t, resp2.Header.Peek(http.HeaderSetCookie))
	fasthttp.ReleaseResponse(resp2)
	qt.Check(t, qt.Equals(string(cleared.Key()), "__Host-flash"))
	qt.Check(t, qt.IsTrue(cleared.Expire().Before(fasthttp.CookieExpireDelete.Add(1))))

	// A second GET sees nothing and sets no cookie.
	resp3, err := tc.Get("/login")
	defer fasthttp.ReleaseResponse(resp3)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(body(t, resp3), "empty"))
	qt.Check(t, qt.HasLen(resp3.Header.Peek(http.HeaderSetCookie), 0))
}

func TestFlashUnusedSetsNoCookie(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/", func(ctx *Context) {
		ctx.Text("plain")
	})

	resp, err := a.TestClient().Get("/")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.HasLen(resp.Header.Peek(http.HeaderSetCookie), 0))
}

func TestFlashForgedCookieYieldsNothing(t *testing.T) {
	a := flashApp(t)
	tc := a.TestClient()

	resp, err := tc.Get("/login", tc.WithCookie("__Host-flash", "not-a-real-id"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(body(t, resp), "empty"))

	// The stale cookie is cleared.
	cleared := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	qt.Check(t, qt.Equals(string(cleared.Key()), "__Host-flash"))
}

func TestFlashKeepAndNow(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Post("/step1", func(ctx *Context) {
		ctx.Flash.Success("Saved.")
		ctx.Redirect("/step2")
	})
	// step2 re-flashes what it received and adds its own message.
	a.Get("/step2", func(ctx *Context) {
		ctx.Flash.Keep()
		ctx.Flash.Success("Almost done.")
		ctx.Redirect("/step3")
	})
	a.Get("/step3", func(ctx *Context) {
		qt.Check(t, qt.DeepEquals(ctx.Flash.Successes(), []string{"Saved.", "Almost done."}))
		ctx.Text("done")
	})
	// Now exposes this request's own writes without a redirect.
	a.Get("/inline", func(ctx *Context) {
		ctx.Flash.Now()
		ctx.Flash.FieldError("email", "Required.")
		qt.Check(t, qt.Equals(ctx.Flash.FieldErrorFor("email"), "Required."))
		qt.Check(t, qt.IsTrue(ctx.Flash.Has()))
		ctx.Text("inline")
	})

	tc := a.TestClient()

	resp, err := tc.PostForm("/step1", nil)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusSeeOther))
	fasthttp.ReleaseResponse(resp)

	resp, err = tc.Get("/step2")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusFound))
	fasthttp.ReleaseResponse(resp)

	resp, err = tc.Get("/step3")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(body(t, resp), "done"))
	fasthttp.ReleaseResponse(resp)

	// Now still persists the writes for the next request too.
	resp, err = tc.Get("/inline")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(body(t, resp), "inline"))
	qt.Check(t, qt.IsTrue(len(resp.Header.Peek(http.HeaderSetCookie)) > 0))
	fasthttp.ReleaseResponse(resp)
}

func TestFlashConfigCookieNameAndTTL(t *testing.T) {
	a := NewTestApp()
	a.Config().Flash.CookieName = "notice"
	a.Config().Server.IdleTimeout = 100 * time.Second
	a.Start(t)
	defer a.Stop()

	a.Post("/", func(ctx *Context) {
		ctx.Flash.Info("hi")
		ctx.Redirect("/")
	})
	a.Get("/", func(ctx *Context) {
		ctx.Text(ctx.Flash.Infos()[0])
	})

	tc := a.TestClient()

	resp, err := tc.PostForm("/", nil)
	qt.Assert(t, qt.IsNil(err))

	c := parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.Equals(string(c.Key()), "__Host-notice"))
	// IdleTimeout + 10%.
	qt.Check(t, qt.Equals(c.MaxAge(), 110))

	resp, err = tc.Get("/")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(body(t, resp), "hi"))
	fasthttp.ReleaseResponse(resp)

	// An explicit TTL wins over the derived default.
	a.Config().Flash.TTL = 7 * time.Second

	resp, err = tc.PostForm("/", nil)
	qt.Assert(t, qt.IsNil(err))

	c = parseCookie(t, resp.Header.Peek(http.HeaderSetCookie))
	fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.Equals(c.MaxAge(), 7))
}
