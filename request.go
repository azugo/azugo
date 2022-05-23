package azugo

import (
	"net"

	"azugo.io/azugo/internal/utils"

	"github.com/valyala/fasthttp"
)

type Context struct {
	noCopy noCopy //nolint:unused,structcheck

	// Base fastHTTP request context
	context *fasthttp.RequestCtx

	method string // HTTP method
	path   string // HTTP path with the modifications by the configuration -> string copy from pathBuffer

	app *App

	// Header access methods
	Header Header
	// Query access methods
	Query Query
	// Body access methods
	Body Body
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
	} else {
		ctx = v.(*Context)
	}

	// Set method
	if c != nil {
		ctx.method = utils.B2S(c.Request.Header.Method())
	}
	ctx.path = path

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

func (ctx *Context) reset() {
	ctx.context = nil
}

// App returns the application.
func (ctx *Context) App() *App {
	return ctx.app
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

// Path returns the path part of the request URL.
func (ctx *Context) Path() string {
	return ctx.path
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
func (ctx *Context) SetUserValue(name string, value interface{}) {
	ctx.context.SetUserValue(name, value)
}

// UserValue returns the value stored via SetUserValue under the given key.
func (ctx *Context) UserValue(name string) interface{} {
	return ctx.context.UserValue(name)
}
