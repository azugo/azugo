package azugo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestUser(t *testing.T) {
	app := NewTestApp()

	app.Use(func(h RequestHandler) RequestHandler {
		return func(ctx *Context) {
			user := NewStandardUser()
			user.ID = "Test"
			user.Authorized = true
			user.Claims["firstName"] = ClaimStrings{"John"}
			user.Claims["lastName"] = ClaimStrings{"Doe"}

			ctx.SetUser(user)

			h(ctx)
		}
	})

	app.Get("/test", func(ctx *Context) {
		user := ctx.User()
		assert.NotNil(t, user)
		assert.Equal(t, "John Doe", user.DisplayName())
	})

	app.Start(t)
	defer app.Stop()

	resp, err := app.TestClient().Get("/test")
	require.NoError(t, err)
	defer fasthttp.ReleaseResponse(resp)

	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode())
}
