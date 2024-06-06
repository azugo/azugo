package azugo

import (
	"testing"

	"azugo.io/azugo/token"
	"azugo.io/azugo/user"

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

	qt.Assert(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}
