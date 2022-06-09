package azugo

import (
	"bytes"
	"net"

	"azugo.io/azugo/internal/utils"
	"azugo.io/azugo/paginator"

	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

const defaultPageSize = 20

var (
	protocolHTTP      = []byte("http")
	protocolHTTPS     = []byte("https")
	protocolSeparator = []byte("://")

	headerXForwardedProto = []byte("X-Forwarded-Proto")
	headerXForwardedHost  = []byte("X-Forwarded-Host")

	contentTypeFormURLEncoded    = []byte("application/x-www-form-urlencoded")
	contentTypeMultipartFormData = []byte("multipart/form-data")

	nilArgsValuer formKeyValuer = &nilArgs{}
)

type Context struct {
	noCopy noCopy //nolint:unused,structcheck

	// Base fastHTTP request context
	context *fasthttp.RequestCtx

	method     string // HTTP method
	path       string // HTTP path with the modifications by the configuration -> string copy from pathBuffer
	routerPath string // HTTP path as registered in the router

	app *App

	// Header access methods
	Header Header
	// Query access methods
	Query Query
	// Body access methods
	Body Body
	// Form access methods
	Form Form
	// Route parameters access methods
	Params Params
}

func (a *App) acquireCtx(path string, c *fasthttp.RequestCtx) *Context {
	v := a.ctxPool.Get()
	var ctx *Context
	if v == nil {
		ctx = new(Context)
		ctx.app = a
		ctx.Header.app = a
		ctx.Header.ctx = ctx
		ctx.Query.app = a
		ctx.Query.ctx = ctx
		ctx.Body.app = a
		ctx.Body.ctx = ctx
		ctx.Form.app = a
		ctx.Form.ctx = ctx
		ctx.Form.form = nilArgsValuer
		ctx.Params.app = a
		ctx.Params.ctx = ctx
	} else {
		ctx = v.(*Context)
	}

	// Set method
	if c != nil {
		ctx.method = utils.B2S(c.Request.Header.Method())
		if u := c.Request.URI(); u != nil {
			ctx.path = utils.B2S(u.Path())
		}
	}
	ctx.routerPath = path

	if ctx.method == fasthttp.MethodPost || ctx.method == fasthttp.MethodPut || ctx.method == fasthttp.MethodPatch {
		if bytes.Equal(c.Request.Header.ContentType(), contentTypeFormURLEncoded) {
			ctx.Form.form = &postArgs{
				args: c.Request.PostArgs(),
			}
		} else if bytes.HasPrefix(c.Request.Header.ContentType(), contentTypeMultipartFormData) {
			if form, err := c.Request.MultipartForm(); err == nil {
				ctx.Form.form = &multiPartArgs{
					args: form,
				}
			}
		}
	}

	// Attach base fastHTTP request context
	ctx.context = c

	return ctx
}

func (a *App) releaseCtx(ctx *Context) {
	ctx.reset()
	a.ctxPool.Put(ctx)
}

// RequestHandler must process incoming requests.
//
// RequestHandler must call ctx.TimeoutError() before returning
// if it keeps references to ctx and/or its' members after the return.
// Consider wrapping RequestHandler into TimeoutHandler if response time
// must be limited.
type RequestHandler func(ctx *Context)

// Handler is an adapter to process incoming requests using object method.
type Handler interface {
	Handler(*Context)
}

// Handle allows to use object method that implements Handler interface to
// handle incoming requests.
func Handle(h Handler) RequestHandler {
	return h.Handler
}

func (ctx *Context) reset() {
	ctx.Form.form.Reset(ctx)
	ctx.Form.form = nilArgsValuer
	ctx.context = nil
}

// App returns the application.
func (ctx *Context) App() *App {
	return ctx.app
}

// Log returns the logger.
func (ctx *Context) Log() *zap.Logger {
	return ctx.app.Log()
}

// Env returns the application environment.
func (ctx *Context) Env() Environment {
	return ctx.app.Env()
}

// Context returns *fasthttp.RequestCtx that carries a deadline
// a cancellation signal, and other values across API boundaries.
func (ctx *Context) Context() *fasthttp.RequestCtx {
	return ctx.context
}

// Request return the *fasthttp.Request object
// This allows you to use all fasthttp request methods
// https://godoc.org/github.com/valyala/fasthttp#Request
func (ctx *Context) Request() *fasthttp.Request {
	return &ctx.context.Request
}

// IP returns the client's network IP address.
func (ctx *Context) IP() net.IP {
	t, ok := ctx.context.RemoteAddr().(*net.TCPAddr)
	if !ok || t.IP == nil {
		return nil
	}
	return t.IP
}

// Method returns the request method.
func (ctx *Context) Method() string {
	return ctx.method
}

// IsTLS returns true if the underlying connection is TLS.
//
// If the request comes from trusted proxy it will use X-Forwarded-Proto header.
func (ctx *Context) IsTLS() bool {
	if ctx.IsTrustedProxy() {
		if bytes.Equal(ctx.Request().Header.PeekBytes(headerXForwardedProto), protocolHTTPS) {
			return true
		} else if bytes.Equal(ctx.Request().Header.PeekBytes(headerXForwardedProto), protocolHTTP) {
			return false
		}
	}
	return ctx.context.IsTLS()
}

// Host returns requested host.
//
// If the request comes from trusted proxy it will use X-Forwarded-Host header.
func (ctx *Context) Host() string {
	// Check if custom host is set
	if len(ctx.app.RouterOptions.Host) > 0 {
		return ctx.app.RouterOptions.Host
	}
	// Use proxy set header
	if ctx.IsTrustedProxy() {
		if host := ctx.context.Request.Header.PeekBytes(headerXForwardedHost); len(host) > 0 {
			return utils.B2S(host)
		}
	}
	// Detect from request
	return utils.B2S(ctx.context.Host())
}

// BasePath returns the base path.
func (ctx *Context) BasePath() string {
	return ctx.app.basePath()
}

// BaseURL returns the base URL for the request.
func (ctx *Context) BaseURL() string {
	url := bytebufferpool.Get()
	defer bytebufferpool.Put(url)

	if ctx.IsTLS() {
		_, _ = url.Write(protocolHTTPS)
	} else {
		_, _ = url.Write(protocolHTTP)
	}
	_, _ = url.Write(protocolSeparator)
	_, _ = url.WriteString(ctx.Host())
	_, _ = url.WriteString(ctx.BasePath())

	return url.String()
}

// RouterPath returns the registered router path.
func (ctx *Context) RouterPath() string {
	return ctx.routerPath
}

// Path returns the path part of the request URL.
func (ctx *Context) Path() string {
	return ctx.path
}

// UserAgent returns the client's User-Agent, if sent in the request.
func (ctx *Context) UserAgent() string {
	return utils.B2S(ctx.context.Request.Header.UserAgent())
}

// SetUserValue stores the given value (arbitrary object)
// under the given key in context.
//
// The value stored in contex may be obtained by UserValue.
//
// This functionality may be useful for passing arbitrary values between
// functions involved in request processing.
//
// All the values are removed from context after returning from the top
// RequestHandler. Additionally, Close method is called on each value
// implementing io.Closer before removing the value from context.
func (ctx *Context) SetUserValue(name string, value any) {
	ctx.context.SetUserValue(name, value)
}

// UserValue returns the value stored via SetUserValue under the given key.
func (ctx *Context) UserValue(name string) any {
	return ctx.context.UserValue(name)
}

// Returns Paginator with page size from query parameters
func (ctx *Context) Paging() *paginator.Paginator {
	page, err := ctx.Query.Int(paginator.QueryParameterPage)
	if err != nil || page <= 0 {
		page = 1
	}
	pageSize, err := ctx.Query.Int(paginator.QueryParameterPerPage)
	if err != nil || pageSize <= 0 {
		pageSize = defaultPageSize
	}
	return paginator.New(page*pageSize, pageSize, page)
}
