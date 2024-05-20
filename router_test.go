package azugo

import (
	"fmt"
	"math/rand"
	"testing"

	"azugo.io/azugo/internal/utils"

	"github.com/go-quicktest/qt"
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
func routerLookupRequest(r *mux, method, path string, ctx *fasthttp.RequestCtx) (fasthttp.RequestHandler, bool) {
	methodIndex := r.MethodIndexOf(method)
	if methodIndex == -1 {
		return nil, false
	}

	if tree := r.trees[methodIndex]; tree != nil {
		handler, tsr := tree.Get(path, ctx)
		if handler != nil || tsr {
			return handler, tsr
		}
	}

	if tree := r.trees[r.MethodIndexOf(MethodWild)]; tree != nil {
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

		h, tsr := routerLookupRequest(a.defaultMux, fasthttp.MethodGet, e.path, ctx)

		qt.Check(t, qt.Equals(tsr, e.tsr), qt.Commentf("TSR (path: %s)", e.path))

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
		qt.Check(t, qt.IsTrue(ok), qt.Commentf("wrong wildcard value missing"))
		qt.Check(t, qt.Equals(param, want), qt.Commentf("wrong wildcard value"))

		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get(fmt.Sprintf("/user/%s", want))
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.IsTrue(routed), qt.Commentf("routing failed"))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
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
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(get), qt.Commentf("GET route not handled"))

	resp, err = a.TestClient().Head("/HEAD")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(head), qt.Commentf("HEAD route not handled"))

	resp, err = a.TestClient().Post("/POST", nil)
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(post), qt.Commentf("POST route not handled"))

	resp, err = a.TestClient().Put("/PUT", nil)
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(put), qt.Commentf("PUT route not handled"))

	resp, err = a.TestClient().Patch("/PATCH", nil)
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(patch), qt.Commentf("PATCH route not handled"))

	resp, err = a.TestClient().Delete("/DELETE")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(delete), qt.Commentf("DELETE route not handled"))

	resp, err = a.TestClient().Connect("/CONNECT")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(connect), qt.Commentf("CONNECT route not handled"))

	resp, err = a.TestClient().Options("/OPTIONS")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(options), qt.Commentf("OPTIONS route not handled"))

	resp, err = a.TestClient().Trace("/TRACE")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(trace), qt.Commentf("TRACE route not handled"))

	resp, err = a.TestClient().Get("/Handler")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(handled), qt.Commentf("Handler route not handled"))

	for _, method := range httpMethods {
		resp, err = a.TestClient().Call(method, "/ANY", nil)
		fasthttp.ReleaseResponse(resp)
		qt.Assert(t, qt.IsNil(err))
		qt.Check(t, qt.IsTrue(any), qt.Commentf("ANY route not handled"))
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
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.IsTrue(routed), qt.Commentf("routing failed"))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
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
	qt.Assert(t, qt.IsNil(err))

	qt.Check(t, qt.IsTrue(routed), qt.Commentf("routing failed"))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))
}

func TestRouterInvalidInput(t *testing.T) {
	a := NewTestApp()
	a.Start(t)
	defer a.Stop()

	handle := func(*Context) {}

	qt.Check(t, qt.PanicMatches(func() {
		a.Handle("", "/", handle)
	}, "method must not be empty"))

	qt.Check(t, qt.PanicMatches(func() {
		a.Get("", handle)
	}, "path must begin with '/' in path ''"))

	qt.Check(t, qt.PanicMatches(func() {
		a.Get("noSlashRoot", handle)
	}, "path must begin with '/' in path 'noSlashRoot'"))

	qt.Check(t, qt.PanicMatches(func() {
		a.Get("/", nil)
	}, "handler must not be nil"))
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

	var v1 any
	id.Get("/click", func(ctx *Context) {
		v1 = ctx.UserValue("id")
		ctx.StatusCode(fasthttp.StatusOK)
	})

	resp, err := a.TestClient().Get("/v4/123/click")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(v1, "123"), qt.Commentf("user value not set"))

	resp, err = a.TestClient().Get("/metrics")
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(v1, "123"), qt.Commentf("user value should not change after second call"))
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
	qt.Assert(t, qt.IsTrue(a.defaultMux.treeMutable))

	for _, method := range httpMethods {
		a.Handle(method, "/", handler1)
	}

	for method := range a.defaultMux.trees {
		qt.Check(t, qt.IsTrue(a.defaultMux.trees[method].Mutable), qt.Commentf("router tree should be mutable"))
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
			qt.Check(t, qt.PanicMatches(func() {
				a.Handle(method, route, handler2)
			}, "a .*handler is already registered for path '.*'"))

			h, _ := routerLookupRequest(a.defaultMux, method, route, nil)
			qt.Check(t, qt.IsNotNil(h), qt.Commentf("handler should not be nil"))
			h(nil)
			qt.Check(t, qt.IsTrue(called1), qt.Commentf("handler should not be changed"))
			called1 = false
		}

		a.Mutable(true)

		for _, method := range httpMethods {
			a.Handle(method, route, handler2)

			h, _ := routerLookupRequest(a.defaultMux, method, route, nil)
			qt.Check(t, qt.IsNotNil(h), qt.Commentf("handler should be nil"))
			h(nil)
			qt.Check(t, qt.IsTrue(called2), qt.Commentf("handler should be changed"))
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
		if qt.Check(t, qt.IsNil(err)) {
			qt.Check(t, qt.Equals(resp.StatusCode(), expectedStatusCode))
			qt.Check(t, qt.Equals(string(resp.Header.Peek("Allow")), expectedAllowed))
		}
		fasthttp.ReleaseResponse(resp)
	}

	// path
	checkHandling("/path", "OPTIONS, POST", fasthttp.StatusNoContent)

	resp, err := a.TestClient().Options("/doesnotexist")
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusNotFound))
	fasthttp.ReleaseResponse(resp)

	// add another method
	a.Get("/path", handlerFunc)

	// set a global OPTIONS handler
	a.RouterOptions().GlobalOPTIONS = func(ctx *Context) {
		// Adjust status code to 200
		ctx.StatusCode(fasthttp.StatusOK)
	}

	// path
	checkHandling("/path", "GET, OPTIONS, POST", fasthttp.StatusOK)

	// custom handler
	var custom bool
	a.Options("/path", func(ctx *Context) {
		custom = true
	})

	// test again
	checkHandling("/path", "", fasthttp.StatusOK)
	qt.Check(t, qt.IsTrue(custom), qt.Commentf("custom OPTIONS handler should be called"))
}

func TestRouterNotAllowed(t *testing.T) {
	handlerFunc := func(*Context) {}

	a := NewTestApp()
	a.Post("/path", handlerFunc)

	a.Start(t)
	defer a.Stop()

	checkHandling := func(path, expectedAllowed string, expectedStatusCode int) {
		resp, err := a.TestClient().Get(path)
		if qt.Check(t, qt.IsNil(err)) {
			qt.Check(t, qt.Equals(resp.StatusCode(), expectedStatusCode))
			qt.Check(t, qt.Equals(string(resp.Header.Peek("Allow")), expectedAllowed))
		}
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
	a.RouterOptions().MethodNotAllowed = func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusTeapot)
		ctx.Text(responseText)
	}

	resp, err := a.TestClient().Get("/path")
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusTeapot))
	qt.Check(t, qt.Equals(string(resp.Body()), responseText))
	qt.Check(t, qt.Equals(string(resp.Header.Peek("Allow")), "DELETE, OPTIONS, POST"))
}

func TestRouterPanicHandler(t *testing.T) {
	a := NewTestApp()

	panicHandled := false
	a.RouterOptions().PanicHandler = func(ctx *Context, p any) {
		panicHandled = true
	}

	a.Put("/user/{name}", func(ctx *Context) {
		panic("oops!")
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Put("/user/gopher", nil)
	fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.IsTrue(panicHandled))
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
		qt.Assert(t, qt.IsNil(err))

		qt.Check(t, qt.Equals(resp.StatusCode(), tr.code), qt.Commentf("%s %s: unexpected response status code", reqMethod, tr.route))
		if tr.code != fasthttp.StatusNotFound {
			qt.Check(t, qt.Equals(string(resp.Header.Peek("Location")), tr.location), qt.Commentf("%s %s: unexpected response header", reqMethod, tr.route))
		}
		fasthttp.ReleaseResponse(resp)
	}

	// Test custom not found handler
	var notFound bool
	a.RouterOptions().NotFound = func(ctx *Context) {
		ctx.StatusCode(fasthttp.StatusNotFound)
		notFound = true
	}

	resp, err := a.TestClient().Call(reqMethod, "/nope", nil)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusNotFound))
	qt.Check(t, qt.IsTrue(notFound))
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
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusPermanentRedirect))
	qt.Check(t, qt.Equals(string(resp.Header.Peek("Location")), "http://test/path?key=val"))

	// Test special case where no node for the prefix "/" exists
	a.Get("/a", func(*Context) {})

	resp, err = a.TestClient().Call(fasthttp.MethodPatch, "/", nil)
	defer fasthttp.ReleaseResponse(resp)
	qt.Assert(t, qt.IsNil(err))
	qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusNotFound))
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
		qt.Assert(t, qt.IsNil(err))

		if method == fasthttp.MethodPost {
			qt.Check(t, qt.IsTrue(postFound), qt.Commentf("post handler should be called"))
		} else {
			qt.Check(t, qt.IsTrue(anyFound), qt.Commentf("any handler should be called"))
		}
		qt.Check(t, qt.Equals(resp.StatusCode(), fasthttp.StatusOK))

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
	handle, tsr := routerLookupRequest(a.defaultMux, reqMethod, "/nope", ctx)
	qt.Check(t, qt.IsNil(handle), qt.Commentf("got handle for unregistered pattern: %v", handle))
	qt.Check(t, qt.IsFalse(tsr), qt.Commentf("got wrong TSR recommendation"))

	// insert route and try again
	a.Handle(method, "/user/{name}", wantHandle)
	handle, _ = routerLookupRequest(a.defaultMux, reqMethod, "/user/gopher", ctx)
	qt.Assert(t, qt.IsNotNil(handle))

	handle(ctx)
	qt.Check(t, qt.IsTrue(routed), qt.Commentf("handle should be called"))

	for expectedKey, expectedVal := range wantParams {
		qt.Check(t, qt.Equals(ctx.UserValue(expectedKey).(string), expectedVal), qt.Commentf("values not saved in context"))
	}

	routed = false

	// route without param
	a.Handle(method, "/user", wantHandle)
	handle, _ = routerLookupRequest(a.defaultMux, reqMethod, "/user", ctx)
	qt.Assert(t, qt.IsNotNil(handle))

	handle(ctx)
	qt.Check(t, qt.IsTrue(routed), qt.Commentf("handle should be called"))

	handle, tsr = routerLookupRequest(a.defaultMux, reqMethod, "/user/gopher/", ctx)
	qt.Check(t, qt.IsNil(handle), qt.Commentf("got handle for unregistered pattern: %v", handle))
	qt.Check(t, qt.IsTrue(tsr), qt.Commentf("got wrong TSR recommendation"))

	handle, tsr = routerLookupRequest(a.defaultMux, reqMethod, "/nope", ctx)
	qt.Check(t, qt.IsNil(handle), qt.Commentf("got handle for unregistered pattern: %v", handle))
	qt.Check(t, qt.IsFalse(tsr), qt.Commentf("got wrong TSR recommendation"))
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
		qt.Check(t, qt.Equals(ctx.RouterPath(), route1))
		routed1 = true
	}

	route2 := "/user/{name}/details"
	routed2 := false
	handle2 := func(ctx *Context) {
		qt.Check(t, qt.Equals(ctx.RouterPath(), route2))
		routed2 = true
	}

	route3 := "/"
	routed3 := false
	handle3 := func(ctx *Context) {
		qt.Check(t, qt.Equals(ctx.RouterPath(), route3))
		routed3 = true
	}

	a := NewTestApp()

	a.Get(route1, handle1)
	a.Get(route2, handle2)
	a.Get(route3, handle3)

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/user/gopher")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(routed1))
	qt.Check(t, qt.IsFalse(routed2))
	qt.Check(t, qt.IsFalse(routed3))

	resp, err = a.TestClient().Get("/user/gopher/details")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(routed2))
	qt.Check(t, qt.IsFalse(routed3))

	resp, err = a.TestClient().Get("/")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(routed3))
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

	qt.Check(t, qt.ContentEquals(a.Routes(), expected))
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
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/v1/foo/2/3")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	resp, err = a.TestClient().Get("/v1/foo/v3")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(routed1), qt.Commentf("/foo/{id}/{pageSize}/{page} not routed"))
	qt.Check(t, qt.IsTrue(routed2), qt.Commentf("/foo/{id}/{iid} not routed"))
	qt.Check(t, qt.IsTrue(routed3), qt.Commentf("/foo/{id} not routed"))
	qt.Check(t, qt.Equals(id1, "1"), qt.Commentf("/foo/{id}/{pageSize}/{page} invalid id value received"))
	qt.Check(t, qt.Equals(pageSize, "20"), qt.Commentf("/foo/{id}/{pageSize}/{page} invalid pageSize value received"))
	qt.Check(t, qt.Equals(page, "4"), qt.Commentf("/foo/{id}/{pageSize}/{page} invalid page value received"))
	qt.Check(t, qt.Equals(id2, "2"), qt.Commentf("/foo/{id}/{iid} invalid id value received"))
	qt.Check(t, qt.Equals(iid, "3"), qt.Commentf("/foo/{id}/{iid} invalid iid value received"))
	qt.Check(t, qt.Equals(id3, "v3"), qt.Commentf("/foo/{id} invalid id value received"))
}

func TestRouterMiddlewares(t *testing.T) {
	var middleware1, middleware2, handled bool

	a := NewTestApp()
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware1 = true
			qt.Check(t, qt.IsFalse(middleware2), qt.Commentf("Second middleware should not be called yet"))
			next(ctx)
		}
	})
	a.Use(func(next RequestHandler) RequestHandler {
		return func(ctx *Context) {
			middleware2 = true
			qt.Check(t, qt.IsTrue(middleware1), qt.Commentf("First middleware should already be called"))
			next(ctx)
		}
	})
	a.Get("/", func(ctx *Context) {
		handled = true
	})

	a.Start(t)
	defer a.Stop()

	resp, err := a.TestClient().Get("/")
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(middleware1), qt.Commentf("First middleware not called"))
	qt.Check(t, qt.IsTrue(middleware2), qt.Commentf("Second middleware not called"))
	qt.Check(t, qt.IsTrue(handled), qt.Commentf("Handler not called"))
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
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(middleware1), qt.Commentf("First middleware not called"))
	qt.Check(t, qt.IsFalse(middleware2), qt.Commentf("Second middleware should not be called"))
	qt.Check(t, qt.IsFalse(handled), qt.Commentf("Handler should not be called"))
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
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)

	qt.Check(t, qt.IsTrue(middleware1), qt.Commentf("First middleware not called"))
	qt.Check(t, qt.IsFalse(middleware2), qt.Commentf("Second middleware should not be called"))
	qt.Check(t, qt.IsTrue(handled), qt.Commentf("Handler should be called"))
}

type hostRouteSwitcher struct {
	hosts map[string]RouterHandler
}

func (s hostRouteSwitcher) SelectRouter(ctx *fasthttp.RequestCtx) RouterHandler {
	if handler, ok := s.hosts[utils.B2S(ctx.Host())]; ok {
		return handler
	}
	return nil
}

func TestPerHostRouteSwitcher(t *testing.T) {
	a := NewTestApp()

	r1 := NewRouter(a.App)
	r2 := NewRouter(a.App)

	a.SetRouterSwitch(hostRouteSwitcher{
		hosts: map[string]RouterHandler{
			"host1": r1,
			"host2": r2,
		},
	})

	var host1, host2, hostother bool

	r1.Get("/", func(ctx *Context) {
		host1 = true
	})

	r2.Get("/", func(ctx *Context) {
		host2 = true
	})

	a.Get("/", func(ctx *Context) {
		hostother = true
	})

	a.Start(t)
	defer a.Stop()

	tc := a.TestClient()

	resp, err := tc.Get("/", tc.WithHost("host1"))
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.IsTrue(host1), qt.Commentf("host1 handler not called"))

	resp, err = tc.Get("/", tc.WithHost("host2"))
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.IsTrue(host2), qt.Commentf("host2 handler not called"))

	resp, err = tc.Get("/", tc.WithHost("host3"))
	qt.Assert(t, qt.IsNil(err))
	fasthttp.ReleaseResponse(resp)
	qt.Check(t, qt.IsTrue(hostother), qt.Commentf("default handler not called"))
}
