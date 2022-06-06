package azugo

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

var httpMethods = []string{
	fasthttp.MethodGet,
	fasthttp.MethodHead,
	fasthttp.MethodPost,
	fasthttp.MethodPut,
	fasthttp.MethodPatch,
	fasthttp.MethodDelete,
	fasthttp.MethodConnect,
	fasthttp.MethodOptions,
	fasthttp.MethodTrace,
	MethodWild,
	"CUSTOM",
}

func randomHTTPMethod() string {
	method := httpMethods[rand.Intn(len(httpMethods)-1)]

	for method == MethodWild {
		method = httpMethods[rand.Intn(len(httpMethods)-1)]
	}

	return method
}

// routerLookupRequest allows the manual lookup of a method + path combo.
// If the path was found, it returns the handler function.
// Otherwise the second return value indicates whether a redirection to
// the same path with an extra / without the trailing slash should be performed.
func routerLookupRequest(r *App, method, path string, ctx *fasthttp.RequestCtx) (fasthttp.RequestHandler, bool) {
	methodIndex := r.methodIndexOf(method)
	if methodIndex == -1 {
		return nil, false
	}

	if tree := r.trees[methodIndex]; tree != nil {
		handler, tsr := tree.Get(path, ctx)
		if handler != nil || tsr {
			return handler, tsr
		}
	}

	if tree := r.trees[r.methodIndexOf(MethodWild)]; tree != nil {
		return tree.Get(path, ctx)
	}

	return nil, false
}

func TestGetOptionalPath(t *testing.T) {
	handler := func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusOK)
	}

	expected := []struct {
		path    string
		tsr     bool
		handler RequestHandler
	}{
		{"/show/{name}", false, handler},
		{"/show/{name}/", true, nil},
		{"/show/{name}/{surname}", false, handler},
		{"/show/{name}/{surname}/", true, nil},
		{"/show/{name}/{surname}/at", false, handler},
		{"/show/{name}/{surname}/at/", true, nil},
		{"/show/{name}/{surname}/at/{address}", false, handler},
		{"/show/{name}/{surname}/at/{address}/", true, nil},
		{"/show/{name}/{surname}/at/{address}/{id}", false, handler},
		{"/show/{name}/{surname}/at/{address}/{id}/", true, nil},
		{"/show/{name}/{surname}/at/{address}/{id}/{phone:.*}", false, handler},
		{"/show/{name}/{surname}/at/{address}/{id}/{phone:.*}/", true, nil},
	}

	a := NewTestApp()
	a.Get("/show/{name}/{surname?}/at/{address?}/{id}/{phone?:.*}", handler)
	a.Start(t)
	defer a.Stop()

	for _, e := range expected {
		ctx := new(fasthttp.RequestCtx)

		h, tsr := routerLookupRequest(a.App, fasthttp.MethodGet, e.path, ctx)

		assert.Equal(t, e.tsr, tsr, "TSR (path: %s)", e.path)

		if (e.handler == nil && h != nil) || (e.handler != nil && h == nil) {
			t.Errorf("Handler (path: %s) == %p, want %p", e.path, h, e.handler)
		}
	}
}

func TestRouter(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	routed := false
	want := "gopher"

	a.Get("/user/{name}", func(ctx *Context) {
		routed = true

		param, ok := ctx.UserValue("name").(string)
		assert.True(t, ok, "wrong wildcard value missing")
		assert.Equal(t, want, param, "wrong wildcard value")

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get(fmt.Sprintf("/user/%s", want))
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.True(t, routed, "routing failed")
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestRouterAPI(t *testing.T) {
	var handled, get, head, post, put, patch, delete, connect, options, trace, any bool

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/GET", func(ctx *Context) {
		get = true
	})
	a.Head("/HEAD", func(ctx *Context) {
		head = true
	})
	a.Post("/POST", func(ctx *Context) {
		post = true
	})
	a.Put("/PUT", func(ctx *Context) {
		put = true
	})
	a.Patch("/PATCH", func(ctx *Context) {
		patch = true
	})
	a.Delete("/DELETE", func(ctx *Context) {
		delete = true
	})
	a.Connect("/CONNECT", func(ctx *Context) {
		connect = true
	})
	a.Options("/OPTIONS", func(ctx *Context) {
		options = true
	})
	a.Trace("/TRACE", func(ctx *Context) {
		trace = true
	})
	a.Any("/ANY", func(ctx *Context) {
		any = true
	})
	a.Handle(fasthttp.MethodGet, "/Handler", func(ctx *Context) {
		handled = true
	})

	resp, err := a.TestClient().Get("/GET")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, get, "GET route not handled")

	resp, err = a.TestClient().Head("/HEAD")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, head, "HEAD route not handled")

	resp, err = a.TestClient().Post("/POST", nil)
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, post, "POST route not handled")

	resp, err = a.TestClient().Put("/PUT", nil)
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, put, "PUT route not handled")

	resp, err = a.TestClient().Patch("/PATCH", nil)
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, patch, "PATCH route not handled")

	resp, err = a.TestClient().Delete("/DELETE")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, delete, "DELETE route not handled")

	resp, err = a.TestClient().Connect("/CONNECT")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, connect, "CONNECT route not handled")

	resp, err = a.TestClient().Options("/OPTIONS")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, options, "OPTIONS route not handled")

	resp, err = a.TestClient().Trace("/TRACE")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, trace, "TRACE route not handled")

	resp, err = a.TestClient().Get("/Handler")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, handled, "Handler route not handled")

	for _, method := range httpMethods {
		resp, err = a.TestClient().Call(method, "/ANY", nil)
		fasthttp.ReleaseResponse(resp)
		require.NoError(t, err)
		assert.True(t, any, "ANY route not handled")
		any = false
	}
}

func TestRouterBasePath(t *testing.T) {
	a := NewTestApp()
	a.Config().Server.Path = "/TEST"
	a.Start(t)
	defer a.Stop()

	routed := false

	a.Get("/user", func(ctx *Context) {
		routed = true

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/test/user")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.True(t, routed, "routing failed")
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestRouterBasePathMatchWithStrippedBase(t *testing.T) {
	a := NewTestApp()
	a.Config().Server.Path = "test/"
	a.Start(t)
	defer a.Stop()

	routed := false

	a.Get("/p", func(ctx *Context) {
		routed = true

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/p")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)

	assert.True(t, routed, "routing failed")
	assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "wrong status code")
}

func TestRouterInvalidInput(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	handle := func(*Context) {}

	assert.Panics(t, func() {
		a.Handle("", "/", handle)
	}, "registering empty method did not panic")

	assert.Panics(t, func() {
		a.Get("", handle)
	}, "registering empty path did not panic")

	assert.Panics(t, func() {
		a.Get("noSlashRoot", handle)
	}, "registering path without leading slash did not panic")

	assert.Panics(t, func() {
		a.Get("/", nil)
	}, "registering nil handler did not panic")
}

func TestRouterRegexUserValues(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	a.Get("/metrics", func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusOK)
	})

	v4 := a.Group("/v4")
	id := v4.Group("/{id:^[1-9]\\d*}")

	var v1 interface{}
	id.Get("/click", func(ctx *Context) {
		v1 = ctx.UserValue("id")
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/v4/123/click")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, "123", v1, "user value should be set")

	resp, err = a.TestClient().Get("/metrics")
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, "123", v1, "user value should not change after second call")
}

func TestRouterMutable(t *testing.T) {
	var called1, called2 bool

	handler1 := func(*Context) {
		called1 = true
	}
	handler2 := func(*Context) {
		called2 = true
	}

	a := NewTestApp()
	a.Mutable(true)
	require.True(t, a.treeMutable, "router tree should be mutable")

	for _, method := range httpMethods {
		a.Handle(method, "/", handler1)
	}

	for method := range a.trees {
		assert.True(t, a.trees[method].Mutable, "router tree should be mutable")
	}

	routes := []string{
		"/",
		"/api/{version}",
		"/{filepath:*}",
		"/user{user:.*}",
	}

	for _, route := range routes {
		a = NewTestApp()

		for _, method := range httpMethods {
			a.Handle(method, route, handler1)
		}

		for _, method := range httpMethods {
			assert.Panics(t, func() {
				a.Handle(method, route, handler2)
			}, "registering route for none mutable router did not panic")

			h, _ := routerLookupRequest(a.App, method, route, nil)
			assert.NotNil(t, h, "handler should not be nil")
			h(nil)
			assert.True(t, called1, "handler should not be changed")
			called1 = false
		}

		a.Mutable(true)

		for _, method := range httpMethods {
			a.Handle(method, route, handler2)

			h, _ := routerLookupRequest(a.App, method, route, nil)
			assert.NotNil(t, h, "handler should not be nil")
			h(nil)
			assert.True(t, called2, "handler should be changed")
			called2 = false
		}
	}
}

func TestRouterOPTIONS(t *testing.T) {
	handlerFunc := func(*Context) {}

	a := NewTestApp()
	a.Post("/path", handlerFunc)

	a.Start(t)
	defer a.Stop()

	checkHandling := func(path, expectedAllowed string, expectedStatusCode int) {
		resp, err := a.TestClient().Options(path)
		require.NoError(t, err)
		assert.Equal(t, expectedStatusCode, resp.StatusCode(), "unexpected response status code")
		assert.Equal(t, expectedAllowed, string(resp.Header.Peek("Allow")), "unexpected response header")
		fasthttp.ReleaseResponse(resp)
	}

	// path
	checkHandling("/path", "OPTIONS, POST", fasthttp.StatusOK)

	resp, err := a.TestClient().Options("/doesnotexist")
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, resp.StatusCode(), "unexpected response status code")
	fasthttp.ReleaseResponse(resp)

	// add another method
	a.Get("/path", handlerFunc)

	// set a global OPTIONS handler
	a.RouterOptions.GlobalOPTIONS = func(ctx *Context) {
		// Adjust status code to 204
		ctx.StatusCode(fasthttp.StatusNoContent)
	}

	// path
	checkHandling("/path", "GET, OPTIONS, POST", fasthttp.StatusNoContent)

	// custom handler
	var custom bool
	a.Options("/path", func(ctx *Context) {
		custom = true
	})

	// test again
	checkHandling("/path", "", fasthttp.StatusOK)
	assert.True(t, custom, "custom OPTIONS handler should be called")
}

func TestRouterNotAllowed(t *testing.T) {
	handlerFunc := func(*Context) {}

	a := NewTestApp()
	a.Post("/path", handlerFunc)

	a.Start(t)
	defer a.Stop()

	checkHandling := func(path, expectedAllowed string, expectedStatusCode int) {
		resp, err := a.TestClient().Get(path)
		require.NoError(t, err)
		assert.Equal(t, expectedStatusCode, resp.StatusCode(), "unexpected response status code")
		assert.Equal(t, expectedAllowed, string(resp.Header.Peek("Allow")), "unexpected response header")
	}

	// test not allowed
	checkHandling("/path", "OPTIONS, POST", fasthttp.StatusMethodNotAllowed)

	// add another method
	a.Delete("/path", handlerFunc)
	a.Options("/path", handlerFunc) // must be ignored

	// test again
	checkHandling("/path", "DELETE, OPTIONS, POST", fasthttp.StatusMethodNotAllowed)

	// test custom handler
	responseText := "custom method"
	a.RouterOptions.MethodNotAllowed = func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusTeapot).Text(responseText)
	}

	resp, err := a.TestClient().Get("/path")
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusTeapot, resp.StatusCode(), "unexpected response status code")
	assert.Equal(t, responseText, string(resp.Body()), "unexpected response body")
	assert.Equal(t, "DELETE, OPTIONS, POST", string(resp.Header.Peek("Allow")), "unexpected response header")
}

func TestRouterPanicHandler(t *testing.T) {
	a := NewTestApp()

	panicHandled := false
	a.RouterOptions.PanicHandler = func(ctx *Context, p interface{}) {
		panicHandled = true
	}

	a.Put("/user/{name}", func(ctx *Context) {
		panic("oops!")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Put("/user/gopher", nil)
	fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.True(t, panicHandled, "panic handler should be called")
}

func testRouterNotFoundByMethod(t *testing.T, method string) {
	handlerFunc := func(*Context) {}
	host := "test"

	a := NewTestApp()

	a.Handle(method, "/path", handlerFunc)
	a.Handle(method, "/dir/", handlerFunc)
	a.Handle(method, "/", handlerFunc)
	a.Handle(method, "/{proc}/StaTus", handlerFunc)
	a.Handle(method, "/USERS/{name}/enTRies/", handlerFunc)
	a.Handle(method, "/static/{filepath:*}", handlerFunc)

	a.Start(t)
	defer a.Stop()

	// Moved Permanently, request with GET method
	expectedCode := fasthttp.StatusMovedPermanently
	if method == fasthttp.MethodConnect {
		// CONNECT method does not allow redirects, so Not Found (404)
		expectedCode = fasthttp.StatusNotFound
	} else if method != fasthttp.MethodGet {
		// Permanent Redirect, request with same method
		expectedCode = fasthttp.StatusPermanentRedirect
	}

	type testRoute struct {
		route    string
		code     int
		location string
	}

	testRoutes := []testRoute{
		{"", fasthttp.StatusOK, ""}, // TSR +/ (Not clean by router, this path is cleaned by fasthttp `ctx.Path()`)
		// {"/../path", expectedCode, fmt.Sprintf("http://%s%s", host, "/path")}, // CleanPath (Not clean by router, this path is cleaned by fasthttp `ctx.Path()`)
		{"/nope", fasthttp.StatusNotFound, ""}, // NotFound
	}

	if method != fasthttp.MethodConnect {
		testRoutes = append(testRoutes, []testRoute{
			{"/path/", expectedCode, fmt.Sprintf("http://%s%s", host, "/path")},                                   // TSR -/
			{"/dir", expectedCode, fmt.Sprintf("http://%s%s", host, "/dir/")},                                     // TSR +/
			{"/PATH", expectedCode, fmt.Sprintf("http://%s%s", host, "/path")},                                    // Fixed Case
			{"/DIR/", expectedCode, fmt.Sprintf("http://%s%s", host, "/dir/")},                                    // Fixed Case
			{"/PATH/", expectedCode, fmt.Sprintf("http://%s%s", host, "/path")},                                   // Fixed Case -/
			{"/DIR", expectedCode, fmt.Sprintf("http://%s%s", host, "/dir/")},                                     // Fixed Case +/
			{"/paTh/?name=foo", expectedCode, fmt.Sprintf("http://%s%s", host, "/path?name=foo")},                 // Fixed Case With Query Params +/
			{"/paTh?name=foo", expectedCode, fmt.Sprintf("http://%s%s", host, "/path?name=foo")},                  // Fixed Case With Query Params +/
			{"/sergio/status/", expectedCode, fmt.Sprintf("http://%s%s", host, "/sergio/StaTus")},                 // Fixed Case With Params -/
			{"/users/atreugo/eNtriEs", expectedCode, fmt.Sprintf("http://%s%s", host, "/USERS/atreugo/enTRies/")}, // Fixed Case With Params +/
			{"/STatiC/test.go", expectedCode, fmt.Sprintf("http://%s%s", host, "/static/test.go")},                // Fixed Case Wildcard
		}...)
	}

	reqMethod := method
	if method == MethodWild {
		reqMethod = fasthttp.MethodPut
	}

	for _, tr := range testRoutes {
		resp, err := a.TestClient().Call(reqMethod, tr.route, nil)
		require.NoError(t, err)

		assert.Equal(t, tr.code, resp.StatusCode(), "%s %s: unexpected response status code", reqMethod, tr.route)
		if resp.StatusCode() != fasthttp.StatusNotFound {
			assert.Equal(t, tr.location, string(resp.Header.Peek("Location")), "%s %s: unexpected response header", reqMethod, tr.route)
		}
		fasthttp.ReleaseResponse(resp)
	}

	// Test custom not found handler
	var notFound bool
	a.RouterOptions.NotFound = func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusNotFound)
		notFound = true
	}

	resp, err := a.TestClient().Call(reqMethod, "/nope", nil)
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, resp.StatusCode(), "unexpected response status code")
	assert.True(t, notFound, "not found handler should be called")
}

func TestRouterNotFound(t *testing.T) {
	for _, method := range httpMethods {
		testRouterNotFoundByMethod(t, method)
	}

	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	// Test other method than GET (want 308 instead of 301)
	a.Patch("/path", func(*Context) {})

	resp, err := a.TestClient().Call(fasthttp.MethodPatch, "/path/?key=val", nil)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusPermanentRedirect, resp.StatusCode(), "unexpected response status code")
	assert.Equal(t, "http://test/path?key=val", string(resp.Header.Peek("Location")), "unexpected response header")

	// Test special case where no node for the prefix "/" exists
	a.Get("/a", func(*Context) {})

	resp, err = a.TestClient().Call(fasthttp.MethodPatch, "/", nil)
	defer fasthttp.ReleaseResponse(resp)
	require.NoError(t, err)
	assert.Equal(t, fasthttp.StatusNotFound, resp.StatusCode(), "unexpected response status code")
}

func TestRouterNotFound_MethodWild(t *testing.T) {
	postFound, anyFound := false, false

	a := NewTestApp()

	a.Any("/{path:*}", func(ctx *Context) { anyFound = true })
	a.Post("/specific", func(ctx *Context) { postFound = true })

	for i := 0; i < 100; i++ {
		a.Handle(
			randomHTTPMethod(),
			fmt.Sprintf("/%d", rand.Int63()),
			func(ctx *Context) {},
		)
	}

	a.Start(t)
	defer a.Stop()

	client := a.TestClient()

	for _, method := range httpMethods {
		resp, err := client.Call(method, "/specific", nil)
		require.NoError(t, err)

		if method == fasthttp.MethodPost {
			assert.True(t, postFound, "post handler should be called")
		} else {
			assert.True(t, anyFound, "any handler should be called")
		}
		assert.Equal(t, fasthttp.StatusOK, resp.StatusCode(), "unexpected response status code")

		fasthttp.ReleaseResponse(resp)
		postFound, anyFound = false, false
	}
}

func testRouterLookupByMethod(t *testing.T, method string) {
	reqMethod := method
	if method == MethodWild {
		reqMethod = randomHTTPMethod()
	}

	routed := false
	wantHandle := func(*Context) {
		routed = true
	}
	wantParams := map[string]string{"name": "gopher"}

	ctx := new(fasthttp.RequestCtx)
	a := NewTestApp()

	a.Start(t)
	defer a.Stop()

	// try empty router first
	handle, tsr := routerLookupRequest(a.App, reqMethod, "/nope", ctx)
	assert.Nilf(t, handle, "got handle for unregistered pattern: %v", handle)
	assert.False(t, tsr, "got wrong TSR recommendation")

	// insert route and try again
	a.Handle(method, "/user/{name}", wantHandle)
	handle, _ = routerLookupRequest(a.App, reqMethod, "/user/gopher", ctx)
	require.NotNil(t, handle, "got no handle for registered pattern")

	handle(ctx)
	assert.True(t, routed, "handle should be called")

	for expectedKey, expectedVal := range wantParams {
		assert.Equal(t, expectedVal, ctx.UserValue(expectedKey), "values not saved in context")
	}

	routed = false

	// route without param
	a.Handle(method, "/user", wantHandle)
	handle, _ = routerLookupRequest(a.App, reqMethod, "/user", ctx)
	require.NotNil(t, handle, "got no handle for registered pattern")

	handle(ctx)
	assert.True(t, routed, "handle should be called")

	handle, tsr = routerLookupRequest(a.App, reqMethod, "/user/gopher/", ctx)
	assert.Nilf(t, handle, "got handle for unregistered pattern: %v", handle)
	assert.True(t, tsr, "got no TSR recommendation")

	handle, tsr = routerLookupRequest(a.App, reqMethod, "/nope", ctx)
	assert.Nilf(t, handle, "got handle for unregistered pattern: %v", handle)
	assert.False(t, tsr, "got wrong TSR recommendation")
}

func TestRouterLookup(t *testing.T) {
	for _, method := range httpMethods {
		testRouterLookupByMethod(t, method)
	}
}

func TestRouterMatchedRoutePath(t *testing.T) {
	route1 := "/user/{name}"
	routed1 := false
	handle1 := func(ctx *Context) {
		assert.Equal(t, route1, ctx.RouterPath())
		routed1 = true
	}

	route2 := "/user/{name}/details"
	routed2 := false
	handle2 := func(ctx *Context) {
		assert.Equal(t, route2, ctx.RouterPath())
		routed2 = true
	}

	route3 := "/"
	routed3 := false
	handle3 := func(ctx *Context) {
		assert.Equal(t, route3, ctx.RouterPath())
		routed3 = true
	}

	a := NewTestApp()

	a.Get(route1, handle1)
	a.Get(route2, handle2)
	a.Get(route3, handle3)

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/user/gopher")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, routed1)
	assert.False(t, routed2)
	assert.False(t, routed3)

	resp, err = a.TestClient().Get("/user/gopher/details")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, routed2)
	assert.False(t, routed3)

	resp, err = a.TestClient().Get("/")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, routed3)
}

func TestRoutesList(t *testing.T) {
	expected := map[string][]string{
		"GET":    {"/bar"},
		"PATCH":  {"/foo"},
		"POST":   {"/v1/users/{name}/{surname?}"},
		"DELETE": {"/v1/users/{id?}"},
	}

	a := NewTestApp()
	a.Get("/bar", func(ctx *Context) {})
	a.Patch("/foo", func(ctx *Context) {})

	v1 := a.Group("/v1")
	v1.Post("/users/{name}/{surname?}", func(ctx *Context) {})
	v1.Delete("/users/{id?}", func(ctx *Context) {})

	assert.Equal(t, expected, a.Routes())
}

func TestRouterSamePrefixParamRoute(t *testing.T) {
	var id1, id2, id3, pageSize, page, iid string
	var routed1, routed2, routed3 bool

	a := NewTestApp()
	v1 := a.Group("/v1")
	v1.Get("/foo/{id}/{pageSize}/{page}", func(ctx *Context) {
		id1 = ctx.UserValue("id").(string)
		pageSize = ctx.UserValue("pageSize").(string)
		page = ctx.UserValue("page").(string)
		routed1 = true
	})
	v1.Get("/foo/{id}/{iid}", func(ctx *Context) {
		id2 = ctx.UserValue("id").(string)
		iid = ctx.UserValue("iid").(string)
		routed2 = true
	})
	v1.Get("/foo/{id}", func(ctx *Context) {
		id3 = ctx.UserValue("id").(string)
		routed3 = true
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/v1/foo/1/20/4")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/v1/foo/2/3")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/v1/foo/v3")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, routed1, "/foo/{id}/{pageSize}/{page} not routed")
	assert.True(t, routed2, "/foo/{id}/{iid} not routed")
	assert.True(t, routed3, "/foo/{id} not routed")
	assert.Equal(t, "1", id1, "/foo/{id}/{pageSize}/{page} invalid id value received")
	assert.Equal(t, "20", pageSize, "/foo/{id}/{pageSize}/{page} invalid pageSize value received")
	assert.Equal(t, "4", page, "/foo/{id}/{pageSize}/{page} invalid page value received")
	assert.Equal(t, "2", id2, "/foo/{id}/{iid} invalid id value received")
	assert.Equal(t, "3", iid, "/foo/{id}/{iid} invalid iid value received")
	assert.Equal(t, "v3", id3, "/foo/{id} invalid id value received")
}

func TestRouterMiddlewares(t *testing.T) {
	var middleware1, middleware2, handled bool

	a := NewTestApp()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware1 = true
			assert.False(t, middleware2, "Second middleware should not yet be called")
			next(ctx)
		}
	})
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware2 = true
			assert.True(t, middleware1, "First middleware should already be called")
			next(ctx)
		}
	})
	a.Get("/", func(ctx *Context) {
		handled = true
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, middleware1, "First middleware not called")
	assert.True(t, middleware2, "Second middleware not called")
	assert.True(t, handled, "Handler not called")
}

func TestRouterMiddlewareBlock(t *testing.T) {
	var middleware1, middleware2, handled bool

	a := NewTestApp()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware1 = true
		}
	})
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware2 = true
			next(ctx)
		}
	})
	a.Get("/", func(ctx *Context) {
		handled = true
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, middleware1, "First middleware not called")
	assert.False(t, middleware2, "Second middleware not called")
	assert.False(t, handled, "Handler not called")
}

func TestRouterMiddlewareAfterRoute(t *testing.T) {
	var middleware1, middleware2, handled bool

	a := NewTestApp()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware1 = true
			next(ctx)
		}
	})
	a.Get("/", func(ctx *Context) {
		handled = true
	})
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware2 = true
			next(ctx)
		}
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/")
	require.NoError(t, err)
	fasthttp.ReleaseResponse(resp)

	assert.True(t, middleware1, "First middleware not called")
	assert.False(t, middleware2, "Second middleware should not be called")
	assert.True(t, handled, "Handler not called")
}

func BenchmarkAllowed(b *testing.B) {
	handlerFunc := func(*Context) {}

	a := NewTestApp()
	a.Post("/path", handlerFunc)
	a.Get("/path", handlerFunc)

	b.Run("Global", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = a.allowed("*", fasthttp.MethodOptions)
		}
	})
	b.Run("Path", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = a.allowed("/path", fasthttp.MethodOptions)
		}
	})
}

func BenchmarkRouterGet(b *testing.B) {
	a := NewTestApp()
	a.Get("/hello", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/hello")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Handler(ctx)
	}
}

func BenchmarkRouterParams(b *testing.B) {
	a := NewTestApp()

	a.Get("/{id}", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/hello")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Handler(ctx)
	}
}

func BenchmarkRouterANY(b *testing.B) {
	a := NewTestApp()

	a.Get("/data", func(ctx *Context) {})
	a.Any("/", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Handler(ctx)
	}
}

func BenchmarkRouterGet_ANY(b *testing.B) {
	var (
		resp    = "Bench GET"
		respANY = "Bench GET (ANY)"
	)

	a := NewTestApp()

	a.Get("/", func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusOK).Text(resp)
	})
	a.Any("/", func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusOK).Text(respANY)
	})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("UNICORN")
	ctx.Request.SetRequestURI("/")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Handler(ctx)
	}
}

func BenchmarkRouterNotFound(b *testing.B) {
	a := NewTestApp()

	a.Get("/bench", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/notfound")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Handler(ctx)
	}
}

func BenchmarkRouterFindCaseInsensitive(b *testing.B) {
	a := NewTestApp()

	a.Get("/bench", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/BenCh/.")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Handler(ctx)
	}
}

func BenchmarkRouterRedirectTrailingSlash(b *testing.B) {
	a := NewTestApp()

	a.Get("/bench/", func(ctx *Context) {})

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/bench")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Handler(ctx)
	}
}

func Benchmark_Get(b *testing.B) {
	handler := func(ctx *Context) {}

	a := NewTestApp()

	a.Get("/", handler)
	a.Get("/plaintext", handler)
	a.Get("/json", handler)
	a.Get("/fortune", handler)
	a.Get("/fortune-quick", handler)
	a.Get("/db", handler)
	a.Get("/queries", handler)
	a.Get("/update", handler)

	ctx := new(fasthttp.RequestCtx)
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.SetRequestURI("/update")

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		a.Handler(ctx)
	}
}
