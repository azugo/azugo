package azugo

import (
	"github.com/valyala/fasthttp"
)

// RouteGroup is a sub-router to group paths.
type RouteGroup struct {
	mux         *mux
	middlewares []RequestHandlerFunc
	prefix      string
}

func (g *RouteGroup) chain(handler RequestHandler) RequestHandler {
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		handler = g.middlewares[i](handler)
	}

	return handler
}

// Mutable allows updating the route handler. Sets for all router not only for group.
//
// Disabled by default.
// WARNING: Use with care. It could generate unexpected behaviors.
func (g *RouteGroup) Mutable(v bool) {
	g.mux.Mutable(v)
}

// Group returns a new group.
// Path auto-correction, including trailing slashes, is enabled by default.
func (g *RouteGroup) Group(path string) Router {
	n := &RouteGroup{
		mux:         g.mux,
		prefix:      g.prefix + path,
		middlewares: make([]RequestHandlerFunc, 0),
	}
	n.Use(g.middlewares...)

	return n
}

// Use appends a middleware to the specified route group.
// Middlewares will be executed in the order they were added.
// It will be executed only for the routes that have been
// added after the middleware was registered.
func (g *RouteGroup) Use(middleware ...RequestHandlerFunc) {
	g.middlewares = append(g.middlewares, middleware...)
}

// Handle registers a new request handler with the given path and method.
//
// For GET, POST, PUT, PATCH and DELETE requests the respective shortcut
// functions can be used.
//
// This function is intended for bulk loading and to allow the usage of less
// frequently used, non-standardized or custom methods (e.g. for internal
// communication with a proxy).
func (g *RouteGroup) Handle(method, path string, handler RequestHandler) {
	g.mux.Handle(method, g.prefix+path, g.chain(handler))
}

// Get is a shortcut for HTTP GET method handler.
func (g *RouteGroup) Get(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodGet, path, handler)
}

// Head is a shortcut for HTTP HEAD method handler.
func (g *RouteGroup) Head(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodHead, path, handler)
}

// Post is a shortcut for HTTP POST method handler.
func (g *RouteGroup) Post(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodPost, path, handler)
}

// Put is a shortcut for HTTP PUT method handler.
func (g *RouteGroup) Put(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodPut, path, handler)
}

// Patch is a shortcut for HTTP PATCH method handler.
func (g *RouteGroup) Patch(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodPatch, path, handler)
}

// Delete is a shortcut for HTTP DELETE method handler.
func (g *RouteGroup) Delete(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodDelete, path, handler)
}

// Connect is a shortcut for HTTP CONNECT method handler.
func (g *RouteGroup) Connect(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodConnect, path, handler)
}

// Options is a shortcut for HTTP OPTIONS method handler.
func (g *RouteGroup) Options(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodOptions, path, handler)
}

// Trace is a shortcut for HTTP TRACE method handler.
func (g *RouteGroup) Trace(path string, handler RequestHandler) {
	g.Handle(fasthttp.MethodTrace, path, handler)
}

// Proxy is helper to proxy requests to another host.
func (g *RouteGroup) Proxy(path string, options ...ProxyOption) {
	p := g.mux.newUpstreamProxy(path, options...)
	handler := g.chain(Handle(p))

	g.Any(path, handler)

	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}

	g.Any(path+"{path:*}", handler)
}

// Any is a shortcut for all HTTP methods handler
//
// WARNING: Use only for routes where the request method is not important.
func (g *RouteGroup) Any(path string, handler RequestHandler) {
	g.Handle(MethodWild, path, handler)
}
