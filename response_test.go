package azugo

import (
	"encoding/json"
	"testing"

	"azugo.io/core/paginator"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

func TestResponseJSON(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		user := &testBodyUser{
			Name: "test",
		}
		ctx.JSON(user)
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	var user testBodyUser
	err = json.Unmarshal(resp.Body(), &user)
	require.NoError(t, err)
	assert.Equal(t, "application/json", string(resp.Header.ContentType()))
	assert.Equal(t, "test", user.Name, "wrong response value")
}

func TestResponseContentType(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.
			ContentType("application/xml", "UTF-8").
			Raw([]byte("<test></test>"))
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, "application/xml; charset=UTF-8", string(resp.Header.ContentType()))
	assert.Equal(t, "<test></test>", string(resp.Body()), "wrong response value")
}

func TestResponsePaging(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user/{name}", func(ctx *Context) {
		p := ctx.Paging()
		name := ctx.UserValue("name").(string)

		p = paginator.New(100, p.PageSize(), p.Current())

		ctx.SetPaging(map[string]string{
			"name": name,
		}, p)

		ctx.StatusCode(fasthttp.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user/test")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, "100", string(resp.Header.Peek(HeaderTotalCount)))
	assert.Equal(t, `<http://test/user/test?page=2&per_page=20>; rel="next",<http://test/user/test?page=5&per_page=20>; rel="last"`, string(resp.Header.Peek(HeaderLink)))
	assert.Equal(t, "X-Total-Count, Link", string(resp.Header.Peek(HeaderAccessControlExposeHeaders)))
}

func TestResponseRedirect(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Redirect("http://test/")
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusFound, resp.StatusCode())
	assert.Equal(t, "http://test/", string(resp.Header.Peek("Location")))
}

func TestResponseRedirectStatusCode(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusPermanentRedirect).Redirect("http://test/")
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.Equal(t, fasthttp.StatusPermanentRedirect, resp.StatusCode())
	assert.Equal(t, "http://test/", string(resp.Header.Peek("Location")))
}
