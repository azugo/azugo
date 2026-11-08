package azugo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"azugo.io/azugo/internal/utils"
	"azugo.io/azugo/user"

	"azugo.io/core"
	"azugo.io/core/http"
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

	headerXForwardedProto = []byte(http.HeaderXForwardedProto)
	headerXForwardedHost  = []byte(http.HeaderXForwardedHost)

	contentTypeFormURLEncoded    = []byte(http.ContentTypeFormURLEncoded)
	contentTypeMultipartFormData = []byte(http.ContentTypeMultipartFormData)

	nilArgsValuer formKeyValuer = &nilArgs{}
	nilRequestID                = ulid.ULID{}
)

// Context holds the state and helpers for a single HTTP request.
type Context struct {
	noCopy noCopy

	// Base fastHTTP request context
	context *fasthttp.RequestCtx
	// reqCtx is the effective request context installed via SetContext.
	reqCtx context.Context
	// reqCtxStop detaches the cancellation bridge installed by SetContext.
	reqCtxStop func() bool

	// bodyLimit overrides the maximum request body size when positive.
	bodyLimit int

	// Request lifecycle state, canceled by releaseCtx so that everything
	// derived from this Context ends with the request
	cancelMu       sync.Mutex
	cancelIdle     *sync.Cond
	doneCh         atomic.Pointer[chan struct{}]
	ctxErr         error
	afterFuncs     map[*func()]struct{}
	gen            uint64
	pendingCancels int
	live           bool
	// touched marks that the lifecycle state above is in use
	touched atomic.Bool

	method       http.Method // HTTP method
	path         string      // HTTP path with the modifications by the configuration -> string copy from pathBuffer
	routerPath   string      // HTTP path as registered in the router
	requestID    ulid.ULID   // Request ID
	requestIDStr string      // Cached string form of the request ID

	// Core data
	app  *App
	mux  *mux
	user User

	// Logger
	loggerCore   *zap.Logger
	loggerFields []zap.Field
	// logger is materialized lazily from loggerCore and loggerFields on first use.
	logger *zap.Logger

	// alwaysHeaders are response headers re-applied after the response is reset
	// to render an error (registered via Header.SetAlways).
	alwaysHeaders []headerEntry

	// Header access methods
	Header HeaderCtx
	// Cookie access methods
	Cookie CookieCtx
	// Flash carries one-shot state to the next request after redirect.
	Flash FlashCtx
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
		ctx.loggerFields = make([]zap.Field, 0, 8)
		ctx.Header.ctx = ctx
		ctx.Cookie.ctx = ctx
		ctx.Flash.ctx = ctx
		ctx.Query.ctx = ctx
		ctx.Body.ctx = ctx
		ctx.Form.ctx = ctx
		ctx.Form.form = nilArgsValuer
		ctx.Params.ctx = ctx
	}

	// Heal lifecycle state left behind by a late Done or AfterFunc call that
	// raced the previous release
	if ctx.touched.Load() {
		a.endCtxLifecycle(ctx)
	}

	// Set method
	if c != nil {
		ctx.method = http.Method(utils.B2S(c.Request.Header.Method()))
		if u := c.Request.URI(); u != nil {
			ctx.path = utils.B2S(u.Path())
		}
	}

	ctx.routerPath = path

	// Generate request ID in place to avoid ULID escaping to the heap; ignore errors
	ms := ulid.Timestamp(time.Now().UTC())
	_ = ctx.requestID.SetTime(ms)
	_ = a.entropy.MonotonicRead(ms, ctx.requestID[6:])

	// Attach base fastHTTP request context
	ctx.context = c

	// Attach mux to request context
	ctx.mux = m

	// Set default user as anonymous
	ctx.user = user.Anonymous{}

	// Attach logger to request context
	_ = ctx.ReplaceLogger(a.Log())

	return ctx
}

func (a *App) releaseCtx(ctx *Context) {
	// Untouched requests skip the lifecycle entirely, healed by the next acquireCtx
	if ctx.touched.Load() {
		a.endCtxLifecycle(ctx)
	}

	ctx.reset()
	a.ctxPool.Put(ctx)
}

// endCtxLifecycle cancels everything derived from the Context and restarts
// its lifecycle state for the next occupant.
func (a *App) endCtxLifecycle(ctx *Context) {
	ctx.cancelMu.Lock()

	// Touches after this point belong to the next occupant or to healing
	ctx.touched.Store(false)

	// Bump the generation
	ctx.gen++

	live := ctx.live
	ctx.live = false

	funcs := ctx.afterFuncs
	ctx.afterFuncs = nil

	if ctx.ctxErr == nil {
		if d := ctx.doneCh.Load(); d != nil {
			close(*d)
		}

		if funcs != nil {
			// Canceled children read Err while the funcs below run
			ctx.ctxErr = context.Canceled
		}
	}

	wait := funcs != nil || ctx.pendingCancels > 0
	if !wait {
		ctx.ctxErr = nil
	}

	ctx.doneCh.Store(nil)
	ctx.cancelMu.Unlock()

	if live {
		a.ctxLiveMu.Lock()
		delete(a.ctxLive, ctx)
		a.ctxLiveMu.Unlock()
	}

	if !wait {
		return
	}

	for f := range funcs {
		(*f)()
	}

	// Wait out concurrently canceling goroutines
	ctx.cancelMu.Lock()

	for ctx.pendingCancels > 0 {
		if ctx.cancelIdle == nil {
			ctx.cancelIdle = sync.NewCond(&ctx.cancelMu)
		}

		ctx.cancelIdle.Wait()
	}

	ctx.ctxErr = nil
	ctx.cancelMu.Unlock()
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
	if c.reqCtxStop != nil {
		c.reqCtxStop()
		c.reqCtxStop = nil
	}

	c.Form.form.Reset(c)
	c.Form.form = nilArgsValuer
	c.Form.once = sync.Once{}
	c.Form.err = nil
	c.bodyLimit = 0
	c.Flash.reset()
	c.user = nil
	c.context = nil
	c.reqCtx = nil
	c.mux = nil
	c.loggerFields = c.loggerFields[:0]
	c.loggerCore = nil
	c.logger = nil
	c.requestID = nilRequestID
	c.requestIDStr = ""
	c.alwaysHeaders = c.alwaysHeaders[:0]
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

// SetReadDeadline sets when reading the rest of the request body times out, replacing the server
// ReadTimeout for this request.
func (c *Context) SetReadDeadline(deadline time.Time) error {
	return c.context.SetReadDeadline(deadline)
}

// SetWriteDeadline sets when writing the response times out, replacing the server WriteTimeout
// for this request.
func (c *Context) SetWriteDeadline(deadline time.Time) error {
	return c.context.SetWriteDeadline(deadline)
}

// Request return the *fasthttp.Request object
// This allows you to use all fasthttp request methods
// https://godoc.org/github.com/valyala/fasthttp#Request
func (c *Context) Request() *fasthttp.Request {
	return &c.context.Request
}

// ID returns the unique request identifier.
func (c *Context) ID() string {
	if c.requestIDStr == "" {
		c.requestIDStr = c.requestID.String()
	}

	return c.requestIDStr
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
func (c *Context) Method() http.Method {
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

// Referer returns the client's Referer, if sent in the request.
func (c *Context) Referer() string {
	return utils.B2S(c.context.Request.Header.Referer())
}

// Protocol returns the HTTP protocol version of the request.
func (c *Context) Protocol() string {
	return utils.B2S(c.context.Request.Header.Protocol())
}

// ContentLength returns the request Content-Length header value.
//
// It may be negative if the request body size is unknown.
func (c *Context) ContentLength() int {
	return c.context.Request.Header.ContentLength()
}

// Time when the request was received by the server.
func (c *Context) Time() time.Time {
	return c.context.Time()
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

// Paging returns a Paginator with page size from query parameters.
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
	h := c.Header.Get(http.HeaderAccept)
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
	h := c.Header.Get(http.HeaderAccept)
	if len(h) == 0 {
		return false
	}

	ctGroup, _, _ := strings.Cut(contentType, "/")

	for len(h) > 0 {
		var part string
		if i := strings.IndexByte(h, ','); i >= 0 {
			part = strings.Trim(h[:i], " ")
			h = h[i+1:]
		} else {
			part = strings.Trim(h, " ")
			h = ""
		}

		// Strip priority
		if i := strings.IndexByte(part, ';'); i >= 0 {
			part = strings.TrimRight(part[:i], " ")
		}

		if part == contentType {
			return true
		}

		// Match "type/*" wildcard
		if strings.HasPrefix(part, ctGroup) && part[len(ctGroup):] == "/*" {
			return true
		}
	}

	return false
}

// SetMaxBodySize overrides the maximum request body size for the current request.
//
// Use with Body.Reader or Body.MultipartReader for large uploads as other
// readers buffer the whole body in memory.
func (c *Context) SetMaxBodySize(n int) {
	c.bodyLimit = n
}

// maxBodySize returns the effective maximum request body size.
func (c *Context) maxBodySize() int {
	if c.bodyLimit > 0 {
		return c.bodyLimit
	}

	if conf := c.app.Config(); conf != nil && conf.Server != nil && conf.Server.MaxRequestBodySize > 0 {
		return conf.Server.MaxRequestBodySize
	}

	return fasthttp.DefaultMaxRequestBodySize
}

// readBody returns the request body enforcing the body size limit.
func (c *Context) readBody() ([]byte, error) {
	req := c.Request()
	if !req.IsBodyStream() {
		if buf := req.Body(); len(buf) <= c.maxBodySize() {
			return buf, nil
		}

		return nil, RequestEntityTooLargeError{}
	}

	r, err := c.limitedBodyStream()
	if err != nil {
		return nil, err
	}

	buf, err := io.ReadAll(r)
	if err != nil {
		if errors.Is(err, RequestEntityTooLargeError{}) {
			return nil, err
		}

		return nil, BadRequestError{"unable to read request body", err}
	}

	req.SetBodyRaw(buf)

	return buf, nil
}

// limitedBodyStream returns the streamed request body limited by the body size limit.
func (c *Context) limitedBodyStream() (io.Reader, error) {
	limit := c.maxBodySize()
	if c.Request().Header.ContentLength() > limit {
		return nil, RequestEntityTooLargeError{}
	}

	return &limitedBody{r: c.Request().BodyStream(), n: int64(limit)}, nil
}

// limitedBody fails with RequestEntityTooLargeError when read past the limit.
type limitedBody struct {
	r io.Reader
	n int64
}

func (l *limitedBody) Read(p []byte) (int, error) {
	if l.n < 0 {
		return 0, RequestEntityTooLargeError{}
	}

	n, err := l.r.Read(p[:min(int64(len(p)), l.n+1)])
	if int64(n) > l.n {
		n, l.n = int(l.n), -1

		return n, RequestEntityTooLargeError{}
	}

	l.n -= int64(n)

	return n, err
}
