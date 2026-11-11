package wsfed

import (
	"testing"

	"azugo.io/azugo"
	"azugo.io/core/http"
	"github.com/go-quicktest/qt"
	"github.com/valyala/fasthttp"
)

func TestReadResponseReturnsFormError(t *testing.T) {
	a := azugo.NewTestApp()
	a.Start(t)
	defer a.Stop()

	var ws WsFederation
	a.Post("/wsfed", func(ctx *azugo.Context) {
		ctx.SetMaxBodySize(4)
		_, err := ws.ReadResponse(ctx)
		if err != nil {
			ctx.Error(err)
			return
		}
		ctx.StatusCode(http.StatusOK)
	})

	resp, err := a.TestClient().PostForm("/wsfed", map[string]any{"wa": "wsignout1.0"})
	qt.Assert(t, qt.IsNil(err))
	defer fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusRequestEntityTooLarge))
}
