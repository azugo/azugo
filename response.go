package azugo

import (
	"net/url"
	"strconv"
	"strings"

	"azugo.io/azugo/paginator"

	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
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

// Redirect redirects the request to a given URL with status code 302 (Found) if other redirect status code
// not set already.
func (ctx *Context) Redirect(url string) {
	if !fasthttp.StatusCodeIsRedirect(ctx.Response().StatusCode()) {
		ctx.StatusCode(fasthttp.StatusFound)
	}
	// TODO: Check if it's safe to redirect to provided URL
	ctx.Header.Set("Location", url)
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

// Raw sets response body, but without copying it.
//
// WARNING: From this point onward the body argument must not be changed.
func (ctx *Context) Raw(data []byte) {
	ctx.Response().SetBodyRaw(data)
}

// Error return the error response. Calls either custom ErrorHandler or default if not specified.
func (ctx *Context) Error(err error) {
	ctx.app.handleError(ctx, err)
}

func (ctx *Context) SetPaging(values map[string]string, paginator *paginator.Paginator) {
	ctx.Header.Set(HeaderTotalCount, strconv.Itoa(paginator.Total()))
	ctx.Header.AppendAccessControlExposeHeaders(HeaderTotalCount)
	route := ctx.RouterPath()
	if len(route) == 0 {
		return
	}
	for k, v := range values {
		route = strings.Replace(route, "{"+k+"}", url.PathEscape(v), 1)
	}
	curl, err := url.Parse(ctx.BaseURL() + route)
	if err != nil {
		ctx.app.Log().Error("Failed to prepare paging header", zap.Error((err)))
		return
	}
	paginator.SetURL(curl)
	links := paginator.Links()
	if len(links) > 0 {
		ctx.Header.Set(HeaderLink, strings.Join(links, ","))
		ctx.Header.AppendAccessControlExposeHeaders(HeaderLink)
	}
}
