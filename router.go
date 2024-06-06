package azugo

import (
	"azugo.io/azugo/config"

	"github.com/valyala/fasthttp"
)

// MethodWild wild HTTP method.
const MethodWild = "*"

const InstrumentationRequest = "http-request"

var (
	contentTypeText = []byte("text/plain; charset=utf-8")
	contentTypeJSON = []byte("application/json")
	questionMark    = byte('?')
)

// RequestHandlerFunc is an adapter to allow to use it as wrapper for RequestHandler.
type RequestHandlerFunc func(h RequestHandler) RequestHandler

// Router to handle multiple methods.
type Router interface {
	// Mutable allows updating the route handler.
	//
	// Disabled by default.
	// WARNING: Use with care. It could generate unexpected behaviors
	Mutable(v bool)

	// Group returns a new group.
	// Path auto-correction, including trailing slashes, is enabled by default.
	Group(path string) Router

	// Use appends a middleware to the router.
	// Middlewares will be executed in the order they were added.
	// It will be executed only for the routes that have been
	// added after the middleware was registered.
	Use(middlewares ...RequestHandlerFunc)

	// Handle registers a new request handler with the given path and method.
	//
	// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
	// functions can be used.
	//
	// This function is intended for bulk loading and to allow the usage of less
	// frequently used, non-standardized or custom methods (e.g. for internal
	// communication with a proxy).
	Handle(method, path string, handler RequestHandler)

	// Get is a shortcut for HTTP GET method handler.
	Get(path string, handler RequestHandler)

	// Head is a shortcut for HTTP HEAD method handler.
	Head(path string, handler RequestHandler)

	// Post is a shortcut for HTTP POST method handler.
	Post(path string, handler RequestHandler)

	// Put is a shortcut for HTTP PUT method handler.
	Put(path string, handler RequestHandler)

	// Patch is a shortcut for HTTP PATCH method handler.
	Patch(path string, handler RequestHandler)

	// Delete is a shortcut for HTTP DELETE method handler.
	Delete(path string, handler RequestHandler)

	// Connect is a shortcut for HTTP CONNECT method handler.
	Connect(path string, handler RequestHandler)

	// Options is a shortcut for HTTP OPTIONS method handler.
	Options(path string, handler RequestHandler)

	// Trace is a shortcut for HTTP TRACE method handler.
	Trace(path string, handler RequestHandler)

	// Proxy is helper to proxy requests to another host.
	Proxy(path string, options ...ProxyOption)

	// Any is a shortcut for all HTTP methods handler.
	//
	// WARNING: Use only for routes where the request method is not important.
	Any(path string, handler RequestHandler)
}

type RouterHandler interface {
	Router

	// Handler for processing incoming requests.
	Handler(ctx *fasthttp.RequestCtx)
}

func NewRouter(app *App) RouterHandler {
	return newMux(app)
}

// RouterOptions allow to configure the router behavior.
type RouterOptions struct {
	// Proxy is the options to describe the trusted proxies.
	Proxy ProxyOptions

	// CorsOptions is the options to describe Cross-Origin Resource Sharing (CORS)
	CORS CORSOptions

	// Host is the hostname to be used for URL generation. If not set
	// it will be automatically detected from the request.
	Host string

	// BasePath is the base path of the router.
	BasePath string

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
	PanicHandler func(*Context, any)
}

func (r *RouterOptions) ApplyConfig(conf *config.Configuration) {
	// Apply base path.
	if len(conf.Server.Path) > 0 {
		r.BasePath = conf.Server.Path
	}
	// Apply CORS configuration.
	if len(conf.CORS.Origins) > 0 {
		r.CORS.SetOrigins(conf.CORS.Origins...)
	}
	// Apply Proxy configuration.
	r.Proxy.Clear().ForwardLimit = conf.Proxy.Limit

	for _, p := range conf.Proxy.Address {
		r.Proxy.Add(p)
	}
}

// Mutable allows updating the route handler
//
// Disabled by default.
// WARNING: Use with care. It could generate unexpected behaviors.
func (a *App) Mutable(v bool) {
	a.defaultMux.Mutable(v)
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
	a.defaultMux.Handle(method, path, handler)
}

// Group returns a new group.
// Path auto-correction, including trailing slashes, is enabled by default.
func (a *App) Group(path string) Router {
	return &RouteGroup{
		mux:         a.defaultMux,
		prefix:      path,
		middlewares: make([]RequestHandlerFunc, 0),
	}
}

// Handler makes the router implement the fasthttp.Handler interface.
func (a *App) Handler(ctx *fasthttp.RequestCtx) {
	a.router.SelectRouter(ctx).Handler(ctx)
}

// Routes returns all registered routes grouped by method.
func (a *App) Routes() map[string][]string {
	return a.defaultMux.Routes()
}

// Use appends a middleware to the router.
// Middlewares will be executed in the order they were added.
// It will be executed only for the routes that have been
// added after the middleware was registered.
func (a *App) Use(middlewares ...RequestHandlerFunc) {
	a.defaultMux.Use(middlewares...)
}

// Get is a shortcut for HTTP GET method handler.
func (a *App) Get(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodGet, path, handler)
}

// Head is a shortcut for HTTP HEAD method handler.
func (a *App) Head(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodHead, path, handler)
}

// Post is a shortcut for HTTP POST method handler.
func (a *App) Post(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodPost, path, handler)
}

// Put is a shortcut for HTTP PUT method handler.
func (a *App) Put(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodPut, path, handler)
}

// Patch is a shortcut for HTTP PATCH method handler.
func (a *App) Patch(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodPatch, path, handler)
}

// Delete is a shortcut for HTTP DELETE method handler.
func (a *App) Delete(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodDelete, path, handler)
}

// Connect is a shortcut for HTTP CONNECT method handler.
func (a *App) Connect(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodConnect, path, handler)
}

// Options is a shortcut for HTTP OPTIONS method handler.
func (a *App) Options(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodOptions, path, handler)
}

// Trace is a shortcut for HTTP TRACE method handler.
func (a *App) Trace(path string, handler RequestHandler) {
	a.Handle(fasthttp.MethodTrace, path, handler)
}

// Proxy is helper to proxy requests to another host.
func (a *App) Proxy(path string, options ...ProxyOption) {
	p := a.defaultMux.newUpstreamProxy(path, options...)
	a.Any(path, Handle(p))

	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}

	a.Any(path+"{path:*}", Handle(p))
}

// Any is a shortcut for all HTTP methods handler
//
// WARNING: Use only for routes where the request method is not important.
func (a *App) Any(path string, handler RequestHandler) {
	a.Handle(MethodWild, path, handler)
}

// RouteSwitcher is used to select a router for a request.
type RouteSwitcher interface {
	// SelectRouter returns a router based on the request.
	//
	// To fallback to default App router return nil.
	SelectRouter(ctx *fasthttp.RequestCtx) RouterHandler
}

type defaultRouter struct {
	*App
}

func (r defaultRouter) SelectRouter(*fasthttp.RequestCtx) RouterHandler {
	return r.defaultMux
}

type customRouter struct {
	*App
	custom RouteSwitcher
}

func (r customRouter) SelectRouter(ctx *fasthttp.RequestCtx) RouterHandler {
	if s := r.custom.SelectRouter(ctx); s != nil {
		return s
	}

	return r.defaultMux
}
