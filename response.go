package azugo

import (
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
)

// Response return the *fasthttp.Response object
// This allows you to use all fasthttp response methods
// https://godoc.org/github.com/valyala/fasthttp#Response
func (ctx *Context) Response() *fasthttp.Response {
	return &ctx.context.Response
}

// StatusCode sets the HTTP status code for the response.
// This method is chainable.
func (ctx *Context) StatusCode(status int) *Context {
	ctx.context.Response.SetStatusCode(status)
	return ctx
}

// ContentType sets the Content-Type header for the response with optionally setting charset if provided.
// This method is chainable.
func (ctx *Context) ContentType(contentType string, charset ...string) *Context {
	if len(charset) > 0 {
		ctx.Response().Header.SetContentType(contentType + "; charset=" + charset[0])
	} else {
		ctx.Response().Header.SetContentType(contentType)
	}
	return ctx
}

// JSON serializes the given struct as JSON and sets it as the response body.
func (ctx *Context) JSON(obj interface{}) {
	ctx.Response().Header.SetContentTypeBytes(contentTypeJSON)
	buf, err := json.Marshal(obj)
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.Response().SetBodyRaw(buf)
}

// Text sets the response body to the given text.
func (ctx *Context) Text(text string) {
	ctx.Response().Header.SetContentTypeBytes(contentTypeText)
	ctx.Response().SetBodyString(text)
}

// Error return the error response. Calls either custom ErrorHandler or default if not specified.
func (ctx *Context) Error(err error) {
	ctx.app.handleError(ctx, err)
}
