package azugo

import (
	"bytes"
	"net"
	"strings"
	"time"

	"azugo.io/azugo/internal/utils"
	"azugo.io/azugo/user"

	"azugo.io/core"
	"azugo.io/core/paginator"
	"github.com/oklog/ulid/v2"
	"github.com/valyala/bytebufferpool"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

var (
	protocolHTTP      = []byte("http")
	protocolHTTPS     = []byte("https")
	protocolSeparator = []byte("://")

	headerXForwardedProto = []byte("X-Forwarded-Proto")
	headerXForwardedHost  = []byte("X-Forwarded-Host")

	contentTypeFormURLEncoded    = []byte("application/x-www-form-urlencoded")
	contentTypeMultipartFormData = []byte("multipart/form-data")

	nilArgsValuer formKeyValuer = &nilArgs{}
	nilRequestID                = ulid.ULID{}
)

type Context struct {
	noCopy noCopy //nolint:unused,structcheck

	// Base fastHTTP request context
	context *fasthttp.RequestCtx

	method     string    // HTTP method
	path       string    // HTTP path with the modifications by the configuration -> string copy from pathBuffer
	routerPath string    // HTTP path as registered in the router
	requestID  ulid.ULID // Request ID

	// Core data
	app  *App
	mux  *mux
	user User

	// Logger
	loggerCore   *zap.Logger
	loggerFields map[string]zap.Field
	logger       *zap.Logger

	// Header access methods
	Header HeaderCtx
	// Query access methods
	Query QueryCtx
	// Body access methods
	Body BodyCtx
	// Form access methods
	Form FormCtx
	// Route parameters access methods
	Params ParamsCtx
}

func (a *App) acquireCtx(m *mux, path string, c *fasthttp.RequestCtx) *Context {
	var ctx *Context

	v := a.ctxPool.Get()
	if v != nil {
		p, ok := v.(*Context)
		if ok {
			ctx = p
		} else {
			a.ctxPool.Put(v)
		}
	}

	if ctx == nil {
		ctx = new(Context)
		ctx.app = a
		ctx.loggerFields = make(map[string]zap.Field, 10)
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

	// Ignore error
	ctx.requestID, _ = ulid.New(ulid.Timestamp(time.Now().UTC()), a.entropy)

	// Attach base fastHTTP request context
	ctx.context = c

	// Attach mux to request context
	ctx.mux = m

	// Set default user as anonymous
	ctx.user = user.Anonymous{}

	// Attach logger to request context
	_ = ctx.ReplaceLogger(a.Log())

	if c != nil {
		ctx.initLoggerFields()
	}

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
	Handler(ctx *Context)
}

// Handle allows to use object method that implements Handler interface to
// handle incoming requests.
func Handle(h Handler) RequestHandler {
	return h.Handler
}

func (c *Context) reset() {
	c.Form.form.Reset(c)
	c.Form.form = nilArgsValuer
	c.user = nil
	c.context = nil
	c.mux = nil
	clear(c.loggerFields)
	c.loggerCore = nil
	c.logger = nil
	c.requestID = nilRequestID
}

// App returns the application.
func (c *Context) App() *App {
	return c.app
}

// Env returns the application environment.
func (c *Context) Env() core.Environment {
	return c.app.Env()
}

// RouterOptions returns the router options.
func (c *Context) RouterOptions() *RouterOptions {
	return c.mux.RouterOptions
}

// Context returns *fasthttp.RequestCtx that carries a deadline
// a cancellation signal, and other values across API boundaries.
func (c *Context) Context() *fasthttp.RequestCtx {
	return c.context
}

// Request return the *fasthttp.Request object
// This allows you to use all fasthttp request methods
// https://godoc.org/github.com/valyala/fasthttp#Request
func (c *Context) Request() *fasthttp.Request {
	return &c.context.Request
}

// ID returns the unique request identifier.
func (c *Context) ID() string {
	return c.requestID.String()
}

// IP returns the client's network IP address.
func (c *Context) IP() net.IP {
	t, ok := c.context.RemoteAddr().(*net.TCPAddr)
	if !ok || t.IP == nil {
		return nil
	}

	return t.IP
}

// Method returns the request method.
func (c *Context) Method() string {
	return c.method
}

// IsTLS returns true if the underlying connection is TLS.
//
// If the request comes from trusted proxy it will use X-Forwarded-Proto header.
func (c *Context) IsTLS() bool {
	if c.IsTrustedProxy() {
		if bytes.Equal(c.Request().Header.PeekBytes(headerXForwardedProto), protocolHTTPS) {
			return true
		} else if bytes.Equal(c.Request().Header.PeekBytes(headerXForwardedProto), protocolHTTP) {
			return false
		}
	}

	return c.context.IsTLS()
}

// Host returns requested host.
//
// If the request comes from trusted proxy it will use X-Forwarded-Host header.
func (c *Context) Host() string {
	// Check if custom host is set
	if host := c.mux.Host(); len(host) > 0 {
		return host
	}

	// Use proxy set header
	if c.IsTrustedProxy() {
		if host := c.context.Request.Header.PeekBytes(headerXForwardedHost); len(host) > 0 {
			return utils.B2S(host)
		}
	}
	// Detect from request
	return utils.B2S(c.context.Host())
}

// BasePath returns the base path.
func (c *Context) BasePath() string {
	return c.mux.BasePath()
}

// BaseURL returns the base URL for the request.
func (c *Context) BaseURL() string {
	url := bytebufferpool.Get()
	defer bytebufferpool.Put(url)

	if c.IsTLS() {
		_, _ = url.Write(protocolHTTPS)
	} else {
		_, _ = url.Write(protocolHTTP)
	}

	_, _ = url.Write(protocolSeparator)
	_, _ = url.WriteString(c.Host())
	_, _ = url.WriteString(c.BasePath())

	return url.String()
}

// RouterPath returns the registered router path.
func (c *Context) RouterPath() string {
	return c.routerPath
}

// Path returns the path part of the request URL.
func (c *Context) Path() string {
	return c.path
}

// UserAgent returns the client's User-Agent, if sent in the request.
func (c *Context) UserAgent() string {
	return utils.B2S(c.context.Request.Header.UserAgent())
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
func (c *Context) SetUserValue(name string, value any) {
	c.context.SetUserValue(name, value)
}

// UserValue returns the value stored via SetUserValue under the given key.
func (c *Context) UserValue(name string) any {
	return c.context.UserValue(name)
}

// MaxPageSize returns the maximum page size.
func (c *Context) MaxPageSize() int {
	if maxPageSize, ok := c.UserValue("__max_page_size").(int); ok {
		return maxPageSize
	}

	return c.App().Config().Paging.MaxPageSize
}

// SetMaxPageSize sets the maximum page size for current context.
// Set to 0 to use the default value from the configuration.
func (c *Context) SetMaxPageSize(maxPageSize int) {
	if maxPageSize <= 0 {
		maxPageSize = c.App().Config().Paging.MaxPageSize
	}

	c.SetUserValue("__max_page_size", maxPageSize)
}

// Returns Paginator with page size from query parameters.
func (c *Context) Paging() *paginator.Paginator {
	page, err := c.Query.Int(paginator.QueryParameterPage)
	if err != nil || page <= 0 {
		page = 1
	}

	pageSize, err := c.Query.Int(paginator.QueryParameterPerPage)
	if err != nil || pageSize <= 0 {
		pageSize = c.App().Config().Paging.DefaultPageSize
	}

	if pageSize > c.MaxPageSize() {
		pageSize = c.MaxPageSize()
	}

	return paginator.New(page*pageSize, pageSize, page)
}

// Accepts checks if provided content type is acceptable for client.
func (c *Context) Accepts(contentType string) bool {
	h := c.Header.Get(HeaderAccept)
	if len(h) == 0 {
		return true
	}

	ctGroup, _, _ := strings.Cut(contentType, "/")
	ctGroup += "/*"

	var (
		part string
		pos  int
	)

	for len(h) > 0 && pos != -1 {
		pos = strings.IndexByte(h, ',')
		if pos != -1 {
			part = strings.Trim(h[:pos], " ")
		} else {
			part = strings.Trim(h, " ")
		}
		// Ignore priority
		if f := strings.IndexByte(part, ';'); f != -1 {
			part = strings.TrimRight(part[:f], " ")
		}

		if part == "*/*" {
			return true
		}

		if part == contentType {
			return true
		}

		if part == ctGroup {
			return true
		}

		if pos != -1 {
			h = h[pos+1:]
		}
	}

	return false
}

// AcceptsExplicit checks if provided content type is explicitly acceptable for client.
func (c *Context) AcceptsExplicit(contentType string) bool {
	h := c.Header.Get(HeaderAccept)
	if len(h) == 0 {
		return false
	}

	ctGroup, _, _ := strings.Cut(contentType, "/")
	ctGroup += "/*"

	var (
		part string
		pos  int
	)

	for len(h) > 0 && pos != -1 {
		pos = strings.IndexByte(h, ',')
		if pos != -1 {
			part = strings.Trim(h[:pos], " ")
		} else {
			part = strings.Trim(h, " ")
		}

		// Ignore priority
		if f := strings.IndexByte(part, ';'); f != -1 {
			part = strings.TrimRight(part[:f], " ")
		}

		if part == contentType {
			return true
		}

		if part == ctGroup {
			return true
		}

		if pos != -1 {
			h = h[pos+1:]
		}
	}

	return false
}
