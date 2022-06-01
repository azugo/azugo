package azugo

// RouteGroup is a sub-router to group paths
type RouteGroup struct {
	app         *App
	middlewares []RequestHandlerFunc
	prefix      string
}

func (g *RouteGroup) chain(handler RequestHandler) RequestHandler {
	for i := len(g.middlewares) - 1; i >= 0; i-- {
		handler = g.middlewares[i](handler)
	}
	return handler
}

// Group returns a new group.
// Path auto-correction, including trailing slashes, is enabled by default.
func (g *RouteGroup) Group(path string) *RouteGroup {
	n := g.app.Group(g.prefix + path)
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
	g.app.Handle(method, g.prefix+path, g.chain(handler))
}

// Get is a shortcut for HTTP GET method handler
func (g *RouteGroup) Get(path string, handler RequestHandler) {
	g.app.Get(g.prefix+path, g.chain(handler))
}

// Head is a shortcut for HTTP HEAD method handler
func (g *RouteGroup) Head(path string, handler RequestHandler) {
	g.app.Head(g.prefix+path, g.chain(handler))
}

// Post is a shortcut for HTTP POST method handler
func (g *RouteGroup) Post(path string, handler RequestHandler) {
	g.app.Post(g.prefix+path, g.chain(handler))
}

// Put is a shortcut for HTTP PUT method handler
func (g *RouteGroup) Put(path string, handler RequestHandler) {
	g.app.Put(g.prefix+path, g.chain(handler))
}

// Patch is a shortcut for HTTP PATCH method handler
func (g *RouteGroup) Patch(path string, handler RequestHandler) {
	g.app.Patch(g.prefix+path, g.chain(handler))
}

// Delete is a shortcut for HTTP DELETE method handler
func (g *RouteGroup) Delete(path string, handler RequestHandler) {
	g.app.Delete(g.prefix+path, g.chain(handler))
}

// Connect is a shortcut for HTTP CONNECT method handler
func (g *RouteGroup) Connect(path string, handler RequestHandler) {
	g.app.Connect(g.prefix+path, g.chain(handler))
}

// Options is a shortcut for HTTP OPTIONS method handler
func (g *RouteGroup) Options(path string, handler RequestHandler) {
	g.app.Options(g.prefix+path, g.chain(handler))
}

// Trace is a shortcut for HTTP TRACE method handler
func (g *RouteGroup) Trace(path string, handler RequestHandler) {
	g.app.Trace(g.prefix+path, g.chain(handler))
}

// Proxy is helper to proxy requests to another host
func (g *RouteGroup) Proxy(path string, options ...ProxyOption) {
	path = g.prefix + path

	p := g.app.newUpstreamProxy(path, options...)
	handler := g.chain(Handle(p))

	g.Any(path, handler)
	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}
	g.Any(path+"{path:*}", handler)
}

// Any is a shortcut for all HTTP methods handler
//
// WARNING: Use only for routes where the request method is not important
func (g *RouteGroup) Any(path string, handler RequestHandler) {
	g.app.Any(g.prefix+path, g.chain(handler))
}
