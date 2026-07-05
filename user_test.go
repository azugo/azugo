package azugo

import (
	"testing"

	"azugo.io/azugo/token"
	"azugo.io/azugo/user"

	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestUser(t *testing.T) {
	app := NewTestApp()

	app.Use(func(h RequestHandler) RequestHandler {
		return func(ctx *Context) {
			user := user.New(map[string]token.ClaimStrings{
				"sub":         {"123"},
				"given_name":  {"John"},
				"family_name": {"Doe"},
			})

			ctx.SetUser(user)

			h(ctx)
		}
	})

	app.Get("/test", func(ctx *Context) {
		user := ctx.User()
		qt.Check(t, qt.IsNotNil(user))
		qt.Check(t, qt.Equals(user.DisplayName(), "John Doe"))
		qt.Check(t, qt.IsTrue(user.Authorized()))
		qt.Check(t, qt.Equals(user.ID(), "123"))
	})

	app.Start(t)
	defer app.Stop()

	resp, err := app.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)

	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))
}

func TestUserNewIdentity(t *testing.T) {
	app := NewTestApp()

	app.Use(func(h RequestHandler) RequestHandler {
		return func(ctx *Context) {
			u := user.NewIdentity("456", "read write:users", map[string]token.ClaimStrings{
				"name":  {"Jane Doe"},
				"email": {"jane@example.com"},
			})
			ctx.SetUser(u)
			h(ctx)
		}
	})

	app.Get("/test", func(ctx *Context) {
		u := ctx.User()
		qt.Check(t, qt.IsNotNil(u))
		qt.Check(t, qt.IsTrue(u.Authorized()))
		qt.Check(t, qt.Equals(u.ID(), "456"))
		qt.Check(t, qt.Equals(u.DisplayName(), "Jane Doe"))
		qt.Check(t, qt.IsTrue(u.HasScope("read")))
		qt.Check(t, qt.IsTrue(u.HasScope("write")))
		qt.Check(t, qt.IsTrue(u.HasScopeLevel("write", "users")))
		qt.Check(t, qt.IsFalse(u.HasScope("delete")))
		qt.Check(t, qt.Equals(u.ClaimValue("email"), "jane@example.com"))
	})

	app.Start(t)
	defer app.Stop()

	resp, err := app.TestClient().Get("/test")
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)

	qt.Assert(t, qt.Equals(resp.StatusCode(), http.StatusOK))
}

func TestDefaultAnonymous(t *testing.T) {
	app := NewTestApp()

	app.MockContext(func(ctx *Context) {
		qt.Check(t, qt.IsFalse(ctx.User().Authorized()))
		qt.Check(t, qt.Equals(ctx.User().ID(), ""))
	})
}
