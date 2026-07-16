package azugo

import (
	"io"
	"net/url"
	"path"
	"strconv"
	"strings"

	"azugo.io/core/http"
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

// Redirect the request to target: either a bare path, prefixed with BasePath() or an absolute
// URL matching the current app base URL.
func (c *Context) Redirect(target string) {
	c.RedirectUnsafe(c.sanitizeRedirect(target))
}

// RedirectUnsafe redirects the request to a given URL with status code 302 (Found) if other redirect
// status code not set already.
func (c *Context) RedirectUnsafe(url string) {
	if !http.StatusCodeIsRedirect(c.Response().StatusCode()) {
		c.StatusCode(http.StatusFound)
	}

	c.Header.Set(http.HeaderLocation, url)
}

// sanitizeRedirect resolves target to a same-origin URL/path safe for Redirect to use.
func (c *Context) sanitizeRedirect(target string) string {
	fallback := c.BasePath() + "/"

	if target == "" || strings.ContainsRune(target, '\\') {
		return fallback
	}

	u, err := url.Parse(target)
	if err != nil {
		return fallback
	}

	if u.Path == "" {
		u.Path = "/"
	} else {
		u.Path = path.Clean(u.Path)
	}

	switch {
	case u.Scheme == "" && u.Host == "" && strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "//"):
		return c.BasePath() + u.String()
	case u.Scheme != "" && u.Host != "" && c.isCurrentOrigin(u):
		return u.String()
	default:
		return fallback
	}
}

// isCurrentOrigin reports whether u's scheme and host match the current request's, as
// reflected by BaseURL().
func (c *Context) isCurrentOrigin(u *url.URL) bool {
	base, err := url.Parse(c.BaseURL())

	return err == nil && strings.EqualFold(u.Scheme, base.Scheme) && strings.EqualFold(u.Host, base.Host)
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

// SetPaging sets pagination headers and metadata on the response.
func (c *Context) SetPaging(values map[string]string, paginator *paginator.Paginator) {
	c.Header.Set(http.HeaderTotalCount, strconv.Itoa(paginator.Total()))
	c.Header.AppendAccessControlExposeHeaders(http.HeaderTotalCount)

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
		c.Header.Set(http.HeaderLink, strings.Join(links, ","))
		c.Header.AppendAccessControlExposeHeaders(http.HeaderLink)
	}
}
