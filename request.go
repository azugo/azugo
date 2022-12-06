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
	"golang.org/x/exp/maps"
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
	v := a.ctxPool.Get()
	var ctx *Context
	if v == nil {
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

	// Ignore error
	ctx.requestID, _ = ulid.New(ulid.Timestamp(time.Now().UTC()), a.entropy)

	// Attach base fastHTTP request context
	ctx.context = c

	// Attach mux to request context
	ctx.mux = m

	// Set default user as anonymous
	ctx.user = user.Anonymous{}

	// Attach logger to request context
	if c != nil {
		ctx.initLoggerFields()
	}
	_ = ctx.ReplaceLogger(a.Log())

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
	ctx.user = nil
	ctx.context = nil
	ctx.mux = nil
	maps.Clear(ctx.loggerFields)
	ctx.loggerCore = nil
	ctx.logger = nil
	ctx.requestID = nilRequestID
}

// App returns the application.
func (ctx *Context) App() *App {
	return ctx.app
}

// Env returns the application environment.
func (ctx *Context) Env() core.Environment {
	return ctx.app.Env()
}

// RouterOptions returns the router options.
func (ctx *Context) RouterOptions() *RouterOptions {
	return ctx.mux.RouterOptions
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

// ID returns the unique request identifier.
func (ctx *Context) ID() string {
	return ctx.requestID.String()
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
	if host := ctx.mux.Host(); len(host) > 0 {
		return host
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
	return ctx.mux.BasePath()
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

// Accepts checks if provided content type is acceptable for client.
func (ctx *Context) Accepts(contentType string) bool {
	h := ctx.Header.Get(HeaderAccept)
	if len(h) == 0 {
		return true
	}

	ctGroup, _, _ := strings.Cut(contentType, "/")
	ctGroup = ctGroup + "/*"

	var part string
	var pos int
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
func (ctx *Context) AcceptsExplicit(contentType string) bool {
	h := ctx.Header.Get(HeaderAccept)
	if len(h) == 0 {
		return false
	}

	ctGroup, _, _ := strings.Cut(contentType, "/")
	ctGroup = ctGroup + "/*"

	var part string
	var pos int
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
