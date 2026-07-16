package azugo

import (
	"encoding/json"
	"testing"

	"azugo.io/core/http"
	"azugo.io/core/paginator"
	"github.com/go-quicktest/qt"
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
	qt.Assert(t, qt.IsNil(err))

	var user testBodyUser
	err = json.Unmarshal(resp.Body(), &user)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(string(resp.Header.ContentType()), http.ContentTypeJSON))
	qt.Check(t, qt.Equals(user.Name, "test"))
}

func TestResponseContentType(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.ContentType(http.ContentTypeXML, "UTF-8")
		ctx.Raw([]byte("<test></test>"))
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(string(resp.Header.ContentType()), "application/xml; charset=UTF-8"))
	qt.Check(t, qt.Equals(string(resp.Body()), "<test></test>"))
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

		ctx.StatusCode(http.StatusOK)
	})

	c := a.TestClient()
	resp, err := c.Get("/user/test")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderTotalCount)), "100"))
	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLink)), `<http://test/user/test?page=2&per_page=20>; rel="next",<http://test/user/test?page=5&per_page=20>; rel="last"`))
	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderAccessControlExposeHeaders)), "X-Total-Count, Link"))
}

func TestResponseRedirect(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Redirect("/next")
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusFound))
	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "/next"))
}

func TestResponseRedirectStatusCode(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.StatusCode(http.StatusPermanentRedirect)
		ctx.Redirect("/next")
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusPermanentRedirect))
	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "/next"))
}

func TestResponseRedirectRejectsOffOriginTargets(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/absolute", func(ctx *Context) { ctx.Redirect("http://evil.example/x") })
	a.Get("/protocol-relative", func(ctx *Context) { ctx.Redirect("//evil.example/x") })
	a.Get("/backslash", func(ctx *Context) { ctx.Redirect(`/\evil.example`) })

	c := a.TestClient()

	for _, path := range []string{"/absolute", "/protocol-relative", "/backslash"} {
		resp, err := c.Get(path)
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "/"), qt.Commentf("path=%s", path))
		fasthttp.ReleaseResponse(resp)
	}
}

func TestResponseRedirectPrefixesBasePath(t *testing.T) {
	t.Setenv("BASE_PATH", "/app")

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Redirect("/dashboard")
	})

	c := a.TestClient()
	resp, err := c.Get("/app/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "/app/dashboard"))
}

func TestResponseRedirectClampsPathTraversal(t *testing.T) {
	t.Setenv("BASE_PATH", "/app")

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Redirect("/../../../otherbasepath")
	})

	c := a.TestClient()
	resp, err := c.Get("/app/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "/app/otherbasepath"))
}

func TestResponseRedirectPreservesQueryAndFragment(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Redirect("/dashboard?tab=1#section")
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "/dashboard?tab=1#section"))
}

func TestResponseRedirectAllowsSameOriginAbsoluteURL(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Redirect(ctx.BaseURL() + "/next")
	})

	tc := a.TestClient()
	resp, err := tc.Get("/user", tc.WithHost("test"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "http://test/next"))
}

func TestResponseRedirectAllowsSameOriginAbsoluteURLWithBasePath(t *testing.T) {
	t.Setenv("BASE_PATH", "/app")

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Redirect(ctx.BaseURL() + "/next")
	})

	tc := a.TestClient()
	resp, err := tc.Get("/app/user", tc.WithHost("test"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "http://test/app/next"))
}

func TestResponseRedirectRejectsCrossOriginAbsoluteURLWithMatchingPath(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.Redirect("http://evil.example" + ctx.BasePath() + "/next")
	})

	resp, err := a.TestClient().Get("/user", a.TestClient().WithHost("test"))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "/"))
}

func TestResponseRedirectUnsafeAllowsAnyTarget(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/user", func(ctx *Context) {
		ctx.RedirectUnsafe("http://test/")
	})

	c := a.TestClient()
	resp, err := c.Get("/user")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.Equals(resp.StatusCode(), http.StatusFound))
	qt.Check(t, qt.Equals(string(resp.Header.Peek(http.HeaderLocation)), "http://test/"))
}
