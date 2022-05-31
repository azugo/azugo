package azugo

import (
	"bytes"
	"crypto/rand"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"strings"

	"azugo.io/azugo/internal/radix"
	"azugo.io/azugo/internal/router"
	"azugo.io/azugo/internal/utils"

	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// MethodWild wild HTTP method
const MethodWild = "*"

var (
	contentTypeText = []byte("text/plain; charset=utf-8")
	contentTypeJSON = []byte("application/json")
	questionMark    = byte('?')

	// MatchedRoutePathParam is the param name under which the path of the matched
	// route is stored, if Router.SaveMatchedRoutePath is set.
	MatchedRoutePathParam string
)

// RequestHandlerFunc is an adapter to allow to use it as wrapper for RequestHandler
type RequestHandlerFunc func(h RequestHandler) RequestHandler

// RouterOptions allow to configure the router behavior
type RouterOptions struct {
	// ProxyOptions is the options to describe the trusted proxies.
	ProxyOptions ProxyOptions

	// Host is the hostname to be used for URL generation. If not set
	// it will be automatically detected from the request.
	Host string

	// BasePath is the base path for router.
	//
	// This is useful when deploying the application in a subdirectory.
	BasePath string

	// If enabled, adds the matched route path onto the ctx.UserValue context
	// before invoking the handler.
	// The matched route path is only added to handlers of routes that were
	// registered when this option was enabled.
	SaveMatchedRoutePath bool

	// Enables automatic redirection if the current route can't be matched but a
	// handler for the path with (without) the trailing slash exists.
	// For example if /foo/ is requested but a route only exists for /foo, the
	// client is redirected to /foo with http status code 301 for GET requests
	// and 308 for all other request methods.
	RedirectTrailingSlash bool

	// If enabled, the router tries to fix the current request path, if no
	// handle is registered for it.
	// First superfluous path elements like ../ or // are removed.
	// Afterwards the router does a case-insensitive lookup of the cleaned path.
	// If a handle can be found for this route, the router makes a redirection
	// to the corrected path with status code 301 for GET requests and 308 for
	// all other request methods.
	// For example /FOO and /..//Foo could be redirected to /foo.
	// RedirectTrailingSlash is independent of this option.
	RedirectFixedPath bool

	// If enabled, the router checks if another method is allowed for the
	// current route, if the current request can not be routed.
	// If this is the case, the request is answered with 'Method Not Allowed'
	// and HTTP status code 405.
	// If no other Method is allowed, the request is delegated to the NotFound
	// handler.
	HandleMethodNotAllowed bool

	// If enabled, the router automatically replies to OPTIONS requests.
	// Custom OPTIONS handlers take priority over automatic replies.
	HandleOPTIONS bool

	// An optional RequestHandler that is called on automatic OPTIONS requests.
	// The handler is only called if HandleOPTIONS is true and no OPTIONS
	// handler for the specific path was set.
	// The "Allowed" header is set before calling the handler.
	GlobalOPTIONS RequestHandler

	// Configurable RequestHandler which is called when no matching route is
	// found. If it is not set, default NotFound is used.
	NotFound RequestHandler

	// Configurable RequestHandler which is called when a request
	// cannot be routed and HandleMethodNotAllowed is true.
	// If it is not set, fasthttp.StatusMethodNotAllowed will be returned.
	// The "Allow" header with allowed request methods is set before the handler
	// is called.
	MethodNotAllowed RequestHandler

	// Configurable http handler that will be called when there is an error.
	// It will be automatically called if any of the Azugo helper response methods
	// encounters an error.
	// If it is not set, error message will be returned for errors that implement
	// SafeError interface, otherwise error will be logged and http error code
	// 500 (Internal Server Error) will be returned.
	ErrorHandler func(*Context, error)

	// Function to handle panics recovered from http handlers.
	// It should be used to generate a error page and return the http error code
	// 500 (Internal Server Error).
	// The handler can be used to keep your server from crashing because of
	// unrecovered panics.
	PanicHandler func(*Context, interface{})
}

// Mutable allows updating the route handler
//
// It's disabled by default
//
// WARNING: Use with care. It could generate unexpected behaviors
func (a *App) Mutable(v bool) {
	a.treeMutable = v

	for i := range a.trees {
		tree := a.trees[i]

		if tree != nil {
			tree.Mutable = v
		}
	}
}

func (a *App) chain(handler RequestHandler) RequestHandler {
	for i := len(a.middlewares) - 1; i >= 0; i-- {
		handler = a.middlewares[i](handler)
	}
	return handler
}

func (a *App) wrapHandler(path string, handler RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		c := a.acquireCtx(path, ctx)
		defer a.releaseCtx(c)
		handler(c)
	}
}

// Handle registers a new request handler with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (a *App) Handle(method, path string, handler RequestHandler) {
	switch {
	case len(method) == 0:
		panic("method must not be empty")
	case len(path) < 1 || path[0] != '/':
		panic("path must begin with '/' in path '" + path + "'")
	case handler == nil:
		panic("handler must not be nil")
	}

	a.registeredPaths[method] = append(a.registeredPaths[method], path)

	methodIndex := a.methodIndexOf(method)
	if methodIndex == -1 {
		tree := radix.New()
		tree.Mutable = a.treeMutable

		a.trees = append(a.trees, tree)
		methodIndex = len(a.trees) - 1
		a.customMethodsIndex[method] = methodIndex
	}

	tree := a.trees[methodIndex]
	if tree == nil {
		tree = radix.New()
		tree.Mutable = a.treeMutable

		a.trees[methodIndex] = tree
		a.globalAllowed = a.allowed("*", "")
	}

	if a.RouterOptions.SaveMatchedRoutePath {
		handler = a.saveMatchedRoutePath(path, handler)
	}

	optionalPaths := router.GetOptionalPaths(path)

	wrappedHandler := a.wrapHandler(path, a.chain(handler))

	// if does not have optional paths, adds the original
	if len(optionalPaths) == 0 {
		tree.Add(path, wrappedHandler)
	} else {
		for _, p := range optionalPaths {
			tree.Add(p, wrappedHandler)
		}
	}
}

func (a *App) saveMatchedRoutePath(path string, handler RequestHandler) RequestHandler {
	return func(ctx *Context) {
		ctx.SetUserValue(MatchedRoutePathParam, path)
		handler(ctx)
	}
}

func (a *App) allowed(path, reqMethod string) (allow string) {
	allowed := make([]string, 0, 9)

	if path == "*" || path == "/*" { // server-wide{ // server-wide
		// empty method is used for internal calls to refresh the cache
		if reqMethod == "" {
			for method := range a.registeredPaths {
				if method == fasthttp.MethodOptions {
					continue
				}
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		} else {
			return a.globalAllowed
		}
	} else { // specific path
		for method := range a.registeredPaths {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == fasthttp.MethodOptions {
				continue
			}

			handle, _ := a.trees[a.methodIndexOf(method)].Get(path, nil)
			if handle != nil {
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		}
	}

	if len(allowed) > 0 {
		// Add request method to list of allowed methods
		allowed = append(allowed, fasthttp.MethodOptions)

		// Sort allowed methods.
		// sort.Strings(allowed) unfortunately causes unnecessary allocations
		// due to allowed being moved to the heap and interface conversion
		for i, l := 1, len(allowed); i < l; i++ {
			for j := i; j > 0 && allowed[j] < allowed[j-1]; j-- {
				allowed[j], allowed[j-1] = allowed[j-1], allowed[j]
			}
		}

		// return as comma separated list
		return strings.Join(allowed, ", ")
	}
	return
}

func (a *App) methodIndexOf(method string) int {
	switch method {
	case fasthttp.MethodGet:
		return 0
	case fasthttp.MethodHead:
		return 1
	case fasthttp.MethodPost:
		return 2
	case fasthttp.MethodPut:
		return 3
	case fasthttp.MethodPatch:
		return 4
	case fasthttp.MethodDelete:
		return 5
	case fasthttp.MethodConnect:
		return 6
	case fasthttp.MethodOptions:
		return 7
	case fasthttp.MethodTrace:
		return 8
	case MethodWild:
		return 9
	}

	if i, ok := a.customMethodsIndex[method]; ok {
		return i
	}

	return -1
}

func (a *App) recv(path string, ctx *fasthttp.RequestCtx) {
	if rcv := recover(); rcv != nil {
		c := a.acquireCtx(path, ctx)
		defer a.releaseCtx(c)
		a.RouterOptions.PanicHandler(c, rcv)
	}
}

func (a *App) handleError(ctx *Context, err error) {
	if a.RouterOptions.ErrorHandler != nil {
		a.RouterOptions.ErrorHandler(ctx, err)
	} else {
		// If there is no error, we don't need to do anything
		if err == nil {
			return
		}

		// Check that the error implements method to customize the status code
		rerr, ok := err.(ResponseStatusCode)
		if ok {
			ctx.StatusCode(rerr.StatusCode())
		} else if errors.As(err, &validator.ValidationErrors{}) {
			// Validation errors return a 422 (unprocessable entity) status code
			ctx.StatusCode(fasthttp.StatusUnprocessableEntity)
		} else {
			ctx.StatusCode(fasthttp.StatusInternalServerError)
		}

		// Log the error only if it's server error
		if ctx.Response().StatusCode()/100 == 5 {
			a.Log().Error(err.Error(), zap.Error(err))
		}

		// Check that the error implements method to for safe error message
		resp := NewErrorResponse(err)
		if resp == nil {
			return
		}

		ct := ctx.Response().Header.ContentType()
		if bytes.HasPrefix(ct, []byte("application/json")) {
			data, ierr := json.Marshal(resp)
			if ierr != nil {
				a.Log().Error("error marshalling error response", zap.Error(ierr))
				return
			}
			ctx.Response().SetBodyRaw(data)
		} else if bytes.HasPrefix(ct, []byte("application/xml")) {
			data, ierr := xml.Marshal(resp)
			if ierr != nil {
				a.Log().Error("error marshalling error response", zap.Error(ierr))
				return
			}
			ctx.Response().SetBodyRaw(data)
		} else {
			ctx.Response().SetBodyString(resp.Errors[0].Message)
		}
	}
}

func (a *App) internalRedirect(path string, ctx *fasthttp.RequestCtx, uri []byte, code int) {
	a.wrapHandler(path, a.chain(func(c *Context) {
		c.context.RedirectBytes(uri, code)
	}))(ctx)
}

func (a *App) tryRedirect(ctx *fasthttp.RequestCtx, tree *radix.Tree, tsr bool, method, path string) bool {
	// Moved Permanently, request with GET method
	code := fasthttp.StatusMovedPermanently
	if method != fasthttp.MethodGet {
		// Permanent Redirect, request with same method
		code = fasthttp.StatusPermanentRedirect
	}

	if tsr && a.RouterOptions.RedirectTrailingSlash {
		uri := bytebufferpool.Get()

		if len(path) > 1 && path[len(path)-1] == '/' {
			uri.SetString(path[:len(path)-1])
		} else {
			uri.SetString(path)
			_, _ = uri.WriteString("/")
		}

		queryBuf := ctx.URI().QueryString()
		if len(queryBuf) > 0 {
			_ = uri.WriteByte(questionMark)
			_, _ = uri.Write(queryBuf)
		}

		a.internalRedirect(path, ctx, uri.Bytes(), code)

		bytebufferpool.Put(uri)

		return true
	}

	// Try to fix the request path
	if a.RouterOptions.RedirectFixedPath {
		path := utils.B2S(ctx.Request.URI().Path())

		uri := bytebufferpool.Get()
		found := tree.FindCaseInsensitivePath(
			router.CleanPath(path),
			a.RouterOptions.RedirectTrailingSlash,
			uri,
		)

		if found {
			queryBuf := ctx.URI().QueryString()
			if len(queryBuf) > 0 {
				_ = uri.WriteByte(questionMark)
				_, _ = uri.Write(queryBuf)
			}

			a.internalRedirect(path, ctx, uri.Bytes(), code)

			bytebufferpool.Put(uri)

			return true
		}
	}

	return false
}

// Group returns a new group.
// Path auto-correction, including trailing slashes, is enabled by default.
func (a *App) Group(path string) *RouteGroup {
	return &RouteGroup{
		app:         a,
		prefix:      path,
		middlewares: append([]RequestHandlerFunc{}, a.middlewares...),
	}
}

// Handler makes the router implement the fasthttp.Handler interface.
func (a *App) Handler(ctx *fasthttp.RequestCtx) {
	path := ""
	if ctx != nil && ctx.Request.URI() != nil {
		path = utils.B2S(ctx.Request.URI().PathOriginal())
	}

	// Remove base path from request path
	basePath := a.basePath()
	l := len(basePath)
	if l > len(path) {
		l = len(path)
	}
	if l > 0 && strings.EqualFold(basePath, path[:l]) {
		path = path[l:]
		if len(path) == 0 || path[0] != '/' {
			path = "/" + path
		}
	}

	if a.RouterOptions.PanicHandler != nil {
		defer a.recv(path, ctx)
	}

	method := utils.B2S(ctx.Request.Header.Method())
	methodIndex := a.methodIndexOf(method)

	if methodIndex > -1 {
		if tree := a.trees[methodIndex]; tree != nil {
			if handler, tsr := tree.Get(path, ctx); handler != nil {
				handler(ctx)
				return
			} else if method != fasthttp.MethodConnect && path != "/" {
				if ok := a.tryRedirect(ctx, tree, tsr, method, path); ok {
					return
				}
			}
		}
	}

	// Try to search in the wild method tree
	if tree := a.trees[a.methodIndexOf(MethodWild)]; tree != nil {
		if handler, tsr := tree.Get(path, ctx); handler != nil {
			handler(ctx)
			return
		} else if method != fasthttp.MethodConnect && path != "/" {
			if ok := a.tryRedirect(ctx, tree, tsr, method, path); ok {
				return
			}
		}
	}

	if a.RouterOptions.HandleOPTIONS && method == fasthttp.MethodOptions {
		// Handle OPTIONS requests

		if allow := a.allowed(path, fasthttp.MethodOptions); allow != "" {
			ctx.Response.Header.Set("Allow", allow)
			if a.RouterOptions.GlobalOPTIONS != nil {
				a.wrapHandler(path, a.chain(a.RouterOptions.GlobalOPTIONS))(ctx)
			}
			return
		}
	} else if a.RouterOptions.HandleMethodNotAllowed {
		// Handle 405

		if allow := a.allowed(path, method); allow != "" {
			ctx.Response.Header.Set("Allow", allow)
			if a.RouterOptions.MethodNotAllowed != nil {
				a.wrapHandler(path, a.chain(a.RouterOptions.MethodNotAllowed))(ctx)
			} else {
				// TODO: move as default value?
				a.wrapHandler(path, a.chain(func(c *Context) {
					c.StatusCode(fasthttp.StatusMethodNotAllowed).Text(fasthttp.StatusMessage(fasthttp.StatusMethodNotAllowed))
				}))(ctx)
			}
			return
		}
	}

	// Handle 404
	if a.RouterOptions.NotFound != nil {
		a.wrapHandler(path, a.chain(a.RouterOptions.NotFound))(ctx)
	} else {
		// TODO: move as default value?
		a.wrapHandler(path, a.chain(func(c *Context) {
			c.StatusCode(fasthttp.StatusNotFound).Text(fasthttp.StatusMessage(fasthttp.StatusNotFound))
		}))(ctx)
	}
}

// Routes returns all registered routes grouped by method
func (a *App) Routes() map[string][]string {
	return a.registeredPaths
}

// Use appends a middleware to the router.
// Middlewares will be executed in the order they were added.
// It will be executed only for the routes that have been
// added after the middleware was registered.
func (a *App) Use(middlewares ...RequestHandlerFunc) {
	a.middlewares = append(a.middlewares, middlewares...)
}

// Get is a shortcut for HTTP GET method handler
func (a *App) Get(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodGet, path, handler)
}

// Head is a shortcut for HTTP HEAD method handler
func (a *App) Head(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodHead, path, handler)
}

// Post is a shortcut for HTTP POST method handler
func (a *App) Post(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodPost, path, handler)
}

// Put is a shortcut for HTTP PUT method handler
func (a *App) Put(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodPut, path, handler)
}

// Patch is a shortcut for HTTP PATCH method handler
func (a *App) Patch(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodPatch, path, handler)
}

// Delete is a shortcut for HTTP DELETE method handler
func (a *App) Delete(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodDelete, path, handler)
}

// Connect is a shortcut for HTTP CONNECT method handler
func (a *App) Connect(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodConnect, path, handler)
}

// Options is a shortcut for HTTP OPTIONS method handler
func (a *App) Options(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodOptions, path, handler)
}

// Trace is a shortcut for HTTP TRACE method handler
func (a *App) Trace(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodTrace, path, handler)
}

// Proxy is helper to proxy requests to another host
func (a *App) Proxy(path string, options ...ProxyOption) {
	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}
	p := newUpstreamProxy(path, options...)
	a.Any(path+"{path:*}", Handle(p))
	a.Any(path, Handle(p))
}

// Any is a shortcut for all HTTP methods handler
//
// WARNING: Use only for routes where the request method is not important
func (a *App) Any(path string, handler RequestHandler) {
	a.Handle(MethodWild, path, handler)
}

func init() {
	r := make([]byte, 15)
	if _, err := io.ReadFull(rand.Reader, r); err != nil {
		panic(err)
	}
	MatchedRoutePathParam = fmt.Sprintf("__matchedRoutePath::%s__", r)
}
