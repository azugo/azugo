package azugo

import (
	"bytes"
	"encoding/xml"
	"errors"
	"strings"
	"sync"

	"azugo.io/azugo/internal/radix"
	"azugo.io/azugo/internal/router"
	"azugo.io/azugo/internal/utils"

	"azugo.io/core/http"
	"github.com/go-playground/validator/v10"
	"github.com/goccy/go-json"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

type mux struct {
	app *App

	// Routing tree
	trees              []*radix.Tree
	treeMutable        bool
	customMethodsIndex map[string]int
	registeredPaths    map[string][]string
	// Router middlewares
	middlewares []RequestHandlerFunc
	// Cached value of global (*) allowed methods
	globalAllowed string

	// Pointer to the originally set base path in RouterOptions
	originalBasePath *string
	// Cached value of base path
	fixedBasePath string
	pathLock      sync.RWMutex

	// Router options
	RouterOptions *RouterOptions
}

func newMux(app *App) *mux {
	return &mux{
		app: app,

		trees:              make([]*radix.Tree, 10),
		customMethodsIndex: make(map[string]int),
		registeredPaths:    make(map[string][]string),
		middlewares:        make([]RequestHandlerFunc, 0, 10),

		RouterOptions: &RouterOptions{
			Proxy:                  defaultProxyOptions,
			CORS:                   defaultCORSOptions,
			RedirectTrailingSlash:  true,
			RedirectFixedPath:      true,
			HandleMethodNotAllowed: true,
			HandleOPTIONS:          true,
			PanicHandler: func(ctx *Context, err any) {
				ctx.Log().Error("Unhandled error", zap.Any("error", err))
			},
			GlobalOPTIONS: func(ctx *Context) {
				ctx.StatusCode(fasthttp.StatusNoContent)
			},
		},
	}
}

// Host returns the default host of the router.
func (m *mux) Host() string {
	return m.RouterOptions.Host
}

// BasePath returns base path of the application.
func (m *mux) BasePath() string {
	m.pathLock.RLock()
	defer m.pathLock.RUnlock()

	if m.originalBasePath == nil || *m.originalBasePath != m.RouterOptions.BasePath {
		m.pathLock.RUnlock()
		m.pathLock.Lock()

		m.originalBasePath = &m.RouterOptions.BasePath
		m.fixedBasePath = m.RouterOptions.BasePath
		// Add leading slash
		if len(m.fixedBasePath) > 0 && m.fixedBasePath[0] != '/' {
			m.fixedBasePath = "/" + m.fixedBasePath
		}
		// Strip trailing slash
		if len(m.fixedBasePath) > 0 && m.fixedBasePath[len(m.fixedBasePath)-1] == '/' {
			m.fixedBasePath = m.fixedBasePath[:len(m.fixedBasePath)-1]
		}

		m.pathLock.Unlock()
		m.pathLock.RLock()
	}

	return m.fixedBasePath
}

// Mutable allows updating the route handler
//
// Disabled by default.
// WARNING: Use with care. It could generate unexpected behaviors.
func (m *mux) Mutable(v bool) {
	m.treeMutable = v

	for i := range m.trees {
		tree := m.trees[i]

		if tree != nil {
			tree.Mutable = v
		}
	}
}

// Routes returns all registered routes grouped by method.
func (m *mux) Routes() map[string][]string {
	return m.registeredPaths
}

// Use appends a middleware to the router.
// Middlewares will be executed in the order they were added.
// It will be executed only for the routes that have been
// added after the middleware was registered.
func (m *mux) Use(middlewares ...RequestHandlerFunc) {
	m.middlewares = append(m.middlewares, middlewares...)
}

func (m *mux) Chain(handler RequestHandler) RequestHandler {
	for i := len(m.middlewares) - 1; i >= 0; i-- {
		handler = m.middlewares[i](handler)
	}

	return handler
}

func (m *mux) WrapHandler(path string, handler RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		c := m.app.acquireCtx(m, path, ctx)
		defer m.app.releaseCtx(c)

		finish := m.app.Instrumenter().Observe(c, InstrumentationRequest, path)
		defer finish(nil)

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
func (m *mux) Handle(method, path string, handler RequestHandler) {
	switch {
	case len(method) == 0:
		panic("method must not be empty")
	case len(path) < 1 || path[0] != '/':
		panic("path must begin with '/' in path '" + path + "'")
	case handler == nil:
		panic("handler must not be nil")
	}

	m.registeredPaths[method] = append(m.registeredPaths[method], path)

	methodIndex := m.MethodIndexOf(method)
	if methodIndex == -1 {
		tree := radix.New()
		tree.Mutable = m.treeMutable

		m.trees = append(m.trees, tree)
		methodIndex = len(m.trees) - 1
		m.customMethodsIndex[method] = methodIndex
	}

	tree := m.trees[methodIndex]
	if tree == nil {
		tree = radix.New()
		tree.Mutable = m.treeMutable

		m.trees[methodIndex] = tree
		m.globalAllowed = m.Allowed("*", "")
	}

	optionalPaths := router.GetOptionalPaths(path)

	wrappedHandler := m.WrapHandler(path, m.Chain(handler))

	// if does not have optional paths, adds the original
	if len(optionalPaths) == 0 {
		tree.Add(path, wrappedHandler)
	} else {
		for _, p := range optionalPaths {
			tree.Add(p, wrappedHandler)
		}
	}
}

func (m *mux) Allowed(path, reqMethod string) string {
	allowed := make([]string, 0, 9)

	if path == "*" || path == "/*" { // server-wide{ // server-wide
		// empty method is used for internal calls to refresh the cache
		if reqMethod == "" {
			for method := range m.registeredPaths {
				if method == fasthttp.MethodOptions {
					continue
				}
				// Add request method to list of allowed methods
				allowed = append(allowed, method)
			}
		} else {
			return m.globalAllowed
		}
	} else { // specific path
		for method := range m.registeredPaths {
			// Skip the requested method - we already tried this one
			if method == reqMethod || method == fasthttp.MethodOptions {
				continue
			}

			handle, _ := m.trees[m.MethodIndexOf(method)].Get(path, nil)
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

	return ""
}

func (m *mux) MethodIndexOf(method string) int {
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

	if i, ok := m.customMethodsIndex[method]; ok {
		return i
	}

	return -1
}

func (m *mux) Recv(path string, ctx *fasthttp.RequestCtx) {
	if rcv := recover(); rcv != nil {
		c := m.app.acquireCtx(m, path, ctx)
		defer m.app.releaseCtx(c)
		m.RouterOptions.PanicHandler(c, rcv)
	}
}

func (m *mux) HandleNotFound(ctx *Context) {
	if m.RouterOptions.NotFound != nil {
		m.RouterOptions.NotFound(ctx)

		return
	}

	ctx.Response().Reset()
	ctx.StatusCode(fasthttp.StatusNotFound)
	ctx.Text(fasthttp.StatusMessage(fasthttp.StatusNotFound))
}

func (m *mux) HandleError(ctx *Context, err error) {
	if m.RouterOptions.ErrorHandler != nil {
		// Log debug information about error
		m.app.Log().Debug("calling custom handler for error: "+err.Error(), zap.Error(err))

		m.RouterOptions.ErrorHandler(ctx, err)

		return
	}

	// If there is no error, we don't need to do anything
	if err == nil {
		return
	}

	// Check that the error implements method to customize the status code
	switch rerr, ok := err.(http.ResponseStatusCode); {
	case ok:
		ctx.StatusCode(rerr.StatusCode())
	case errors.As(err, &validator.ValidationErrors{}):
		// Validation errors return a 422 (unprocessable entity) status code
		ctx.StatusCode(fasthttp.StatusUnprocessableEntity)
	default:
		ctx.StatusCode(fasthttp.StatusInternalServerError)
	}

	// Log debug information about error
	m.app.Log().Debug("handling error: "+err.Error(), zap.Error(err))

	// Log the error only if it's server error
	if ctx.Response().StatusCode()/100 == 5 {
		m.app.Log().Error(err.Error(), zap.Error(err))
	}

	// Check that the error implements method to for safe error message
	resp := NewErrorResponse(err)
	if resp == nil {
		return
	}

	ct := ctx.Response().Header.ContentType()
	if hasCT := bytes.HasPrefix(ct, []byte(ContentTypeJSON)); hasCT || ctx.AcceptsExplicit(ContentTypeJSON) {
		data, ierr := json.Marshal(resp)
		if ierr != nil {
			m.app.Log().Error("error marshalling error response", zap.Error(ierr))

			return
		}

		if !hasCT {
			ctx.ContentType(ContentTypeJSON)
		}

		ctx.Response().SetBodyRaw(data)
	} else if hasCT := bytes.HasPrefix(ct, []byte(ContentTypeXML)); hasCT || ctx.AcceptsExplicit(ContentTypeXML) {
		data, ierr := xml.Marshal(resp)
		if ierr != nil {
			m.app.Log().Error("error marshalling error response", zap.Error(ierr))

			return
		}

		if !hasCT {
			ctx.ContentType(ContentTypeXML)
		}

		ctx.Response().SetBodyRaw(data)
	} else {
		ctx.Response().SetBodyString(resp.Errors[0].Message)
	}
}

func (m *mux) InternalRedirect(path string, ctx *fasthttp.RequestCtx, uri []byte, code int) {
	m.WrapHandler(path, m.Chain(func(c *Context) {
		c.context.RedirectBytes(uri, code)
	}))(ctx)
}

func (m *mux) TryRedirect(ctx *fasthttp.RequestCtx, tree *radix.Tree, tsr bool, method, path string) bool {
	// Moved Permanently, request with GET method
	code := fasthttp.StatusMovedPermanently
	if method != fasthttp.MethodGet {
		// Permanent Redirect, request with same method
		code = fasthttp.StatusPermanentRedirect
	}

	if tsr && m.RouterOptions.RedirectTrailingSlash {
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

		m.InternalRedirect(path, ctx, uri.Bytes(), code)

		bytebufferpool.Put(uri)

		return true
	}

	// Try to fix the request path
	if m.RouterOptions.RedirectFixedPath {
		path := utils.B2S(ctx.Request.URI().Path())

		uri := bytebufferpool.Get()
		found := tree.FindCaseInsensitivePath(
			router.CleanPath(path),
			m.RouterOptions.RedirectTrailingSlash,
			uri,
		)

		if found {
			queryBuf := ctx.URI().QueryString()
			if len(queryBuf) > 0 {
				_ = uri.WriteByte(questionMark)
				_, _ = uri.Write(queryBuf)
			}

			m.InternalRedirect(path, ctx, uri.Bytes(), code)

			bytebufferpool.Put(uri)

			return true
		}
	}

	return false
}

// Group returns a new group.
// Path auto-correction, including trailing slashes, is enabled by default.
func (m *mux) Group(path string) Router {
	return &RouteGroup{
		mux:         m,
		prefix:      path,
		middlewares: make([]RequestHandlerFunc, 0),
	}
}

// Get is a shortcut for HTTP GET method handler.
func (m *mux) Get(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodGet, path, handler)
}

// Head is a shortcut for HTTP HEAD method handler.
func (m *mux) Head(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodHead, path, handler)
}

// Post is a shortcut for HTTP POST method handler.
func (m *mux) Post(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodPost, path, handler)
}

// Put is a shortcut for HTTP PUT method handler.
func (m *mux) Put(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodPut, path, handler)
}

// Patch is a shortcut for HTTP PATCH method handler.
func (m *mux) Patch(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodPatch, path, handler)
}

// Delete is a shortcut for HTTP DELETE method handler.
func (m *mux) Delete(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodDelete, path, handler)
}

// Connect is a shortcut for HTTP CONNECT method handler.
func (m *mux) Connect(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodConnect, path, handler)
}

// Options is a shortcut for HTTP OPTIONS method handler.
func (m *mux) Options(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodOptions, path, handler)
}

// Trace is a shortcut for HTTP TRACE method handler.
func (m *mux) Trace(path string, handler RequestHandler) {
	m.Handle(fasthttp.MethodTrace, path, handler)
}

// Proxy is helper to proxy requests to another host.
func (m *mux) Proxy(path string, options ...ProxyOption) {
	p := m.newUpstreamProxy(path, options...)
	m.Any(path, Handle(p))

	if len(path) > 0 && path[len(path)-1] != '/' {
		path += "/"
	}

	m.Any(path+"{path:*}", Handle(p))
}

// Any is a shortcut for all HTTP methods handler
//
// WARNING: Use only for routes where the request method is not important.
func (m *mux) Any(path string, handler RequestHandler) {
	m.Handle(MethodWild, path, handler)
}

// Handler makes the router implement the fasthttp.Handler interface.
func (m *mux) Handler(ctx *fasthttp.RequestCtx) {
	path := ""
	if ctx != nil && ctx.Request.URI() != nil {
		path = utils.B2S(ctx.Request.URI().PathOriginal())
	}

	// Remove base path from request path
	basePath := m.BasePath()

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

	if m.RouterOptions.PanicHandler != nil {
		defer m.Recv(path, ctx)
	}

	method := utils.B2S(ctx.Request.Header.Method())
	methodIndex := m.MethodIndexOf(method)

	if methodIndex > -1 {
		if tree := m.trees[methodIndex]; tree != nil {
			if handler, tsr := tree.Get(path, ctx); handler != nil {
				handler(ctx)

				return
			} else if method != fasthttp.MethodConnect && path != "/" {
				if ok := m.TryRedirect(ctx, tree, tsr, method, path); ok {
					return
				}
			}
		}
	}

	// Try to search in the wild method tree
	if tree := m.trees[m.MethodIndexOf(MethodWild)]; tree != nil {
		if handler, tsr := tree.Get(path, ctx); handler != nil {
			handler(ctx)

			return
		} else if method != fasthttp.MethodConnect && path != "/" {
			if ok := m.TryRedirect(ctx, tree, tsr, method, path); ok {
				return
			}
		}
	}

	if m.RouterOptions.HandleOPTIONS && method == fasthttp.MethodOptions {
		// Handle OPTIONS requests
		if allow := m.Allowed(path, fasthttp.MethodOptions); allow != "" {
			ctx.Response.Header.Set("Allow", allow)

			if m.RouterOptions.GlobalOPTIONS != nil {
				m.WrapHandler(path, m.Chain(m.RouterOptions.GlobalOPTIONS))(ctx)
			}

			return
		}
	} else if m.RouterOptions.HandleMethodNotAllowed {
		// Handle 405
		if allow := m.Allowed(path, method); allow != "" {
			ctx.Response.Header.Set("Allow", allow)

			if m.RouterOptions.MethodNotAllowed != nil {
				m.WrapHandler(path, m.Chain(m.RouterOptions.MethodNotAllowed))(ctx)

				return
			}

			// TODO: move as default value?
			m.WrapHandler(path, m.Chain(func(c *Context) {
				c.StatusCode(fasthttp.StatusMethodNotAllowed)
				c.Text(fasthttp.StatusMessage(fasthttp.StatusMethodNotAllowed))
			}))(ctx)

			return
		}
	}

	// Handle 404
	m.WrapHandler(path, m.Chain(m.HandleNotFound))(ctx)
}

// InstrRequest returns path if the request is router handler request.
func InstrRequest(op string, args ...any) (string, bool) {
	if op != InstrumentationRequest || len(args) != 1 {
		return "", false
	}

	path, ok := args[0].(string)

	return path, ok
}
