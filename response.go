package azugo

import (
	"io"
	"net/url"
	"strconv"
	"strings"

	"azugo.io/core/paginator"
	"github.com/goccy/go-json"
	"github.com/valyala/fasthttp"
	"go.uber.org/zap"
)

// Response return the *fasthttp.Response object
// This allows you to use all fasthttp response methods
// https://godoc.org/github.com/valyala/fasthttp#Response
func (c *Context) Response() *fasthttp.Response {
	return &c.context.Response
}

// StatusCode sets the HTTP status code for the response.
// This method is chainable.
func (c *Context) StatusCode(status int) {
	c.context.Response.SetStatusCode(status)
}

// ContentType sets the Content-Type header for the response with optionally setting charset if provided.
// This method is chainable.
func (c *Context) ContentType(contentType string, charset ...string) {
	if len(charset) > 0 {
		c.Response().Header.SetContentType(contentType + "; charset=" + charset[0])
	} else {
		c.Response().Header.SetContentType(contentType)
	}
}

// Redirect redirects the request to a given URL with status code 302 (Found) if other redirect status code
// not set already.
func (c *Context) Redirect(url string) {
	if !fasthttp.StatusCodeIsRedirect(c.Response().StatusCode()) {
		c.StatusCode(fasthttp.StatusFound)
	}
	// TODO: Check if it's safe to redirect to provided URL
	c.Header.Set("Location", url)
}

// JSON serializes the given struct as JSON and sets it as the response body.
func (c *Context) JSON(obj any) {
	c.Response().Header.SetContentTypeBytes(contentTypeJSON)

	buf, err := json.Marshal(obj)
	if err != nil {
		c.Error(err)

		return
	}

	c.Response().SetBodyRaw(buf)
}

// Text sets the response body to the given text.
func (c *Context) Text(text string) {
	c.Response().Header.SetContentTypeBytes(contentTypeText)
	c.Response().SetBodyString(text)
}

// Stream sets the response body to the given stream.
//
// Close() is called after finishing reading all body data if it implements io.Closer.
func (c *Context) Stream(r io.Reader) {
	c.Response().SetBodyStream(r, -1)
}

// Raw sets response body, but without copying it.
//
// WARNING: From this point onward the body argument must not be changed.
func (c *Context) Raw(data []byte) {
	c.Response().SetBodyRaw(data)
}

// Error return the error response. Calls either custom ErrorHandler or default if not specified.
func (c *Context) Error(err error) {
	c.mux.HandleError(c, err, false)
}

// NotFound returns an not found response. Calls either custom NotFound or default if not specified.
func (c *Context) NotFound() {
	c.Response().Reset()
	c.mux.HandleNotFound(c)
}

func (c *Context) SetPaging(values map[string]string, paginator *paginator.Paginator) {
	c.Header.Set(HeaderTotalCount, strconv.Itoa(paginator.Total()))
	c.Header.AppendAccessControlExposeHeaders(HeaderTotalCount)

	route := c.RouterPath()
	if len(route) == 0 {
		return
	}

	for k, v := range values {
		route = strings.Replace(route, "{"+k+"}", url.PathEscape(v), 1)
	}

	curl, err := url.Parse(c.BaseURL() + route)
	if err != nil {
		c.Log().Error("Failed to prepare paging header", zap.Error((err)))

		return
	}

	paginator.SetURL(curl)

	links := paginator.Links()
	if len(links) > 0 {
		c.Header.Set(HeaderLink, strings.Join(links, ","))
		c.Header.AppendAccessControlExposeHeaders(HeaderLink)
	}
}
