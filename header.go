package azugo

import (
	"bytes"
	"fmt"
	"iter"
	"strings"

	"azugo.io/azugo/internal/utils"

	"azugo.io/core/http"
	"github.com/valyala/fasthttp"
)

// HTTP header name constants.
const (
	HeaderAccept                     string = "Accept"
	HeaderTotalCount                 string = "X-Total-Count"
	HeaderLink                       string = "Link"
	HeaderAccessControlExposeHeaders string = "Access-Control-Expose-Headers"
	HeaderContentType                string = "Content-Type"
	HeaderContentDisposition         string = "Content-Disposition"
	HeaderContentTransferEncoding    string = "Content-Transfer-Encoding"
)

// HTTP content type constants.
const (
	ContentTypeJSON        string = "application/json"
	ContentTypeXML         string = "application/xml"
	ContentTypeOctetStream string = "application/octet-stream"
)

// HeaderCtx represents the key-value pairs in an HTTP header.
type HeaderCtx struct {
	noCopy noCopy

	ctx *Context
}

// Get gets the first value associated with the given key in request.
// If there are no values associated with the key, Get returns "".
func (h *HeaderCtx) Get(key string) string {
	return utils.B2S(h.ctx.Request().Header.Peek(key))
}

// Values returns all values associated with the given key in request.
func (h *HeaderCtx) Values(key string) []string {
	data := make([]string, 0, 1)

	for k, val := range h.ctx.Request().Header.All() {
		if !strings.EqualFold(key, utils.B2S(k)) {
			continue
		}

		if bytes.Contains(val, []byte{','}) {
			values := bytes.Split(val, []byte{','})
			for i := range values {
				data = append(data, utils.B2S(values[i]))
			}
		} else {
			data = append(data, utils.B2S(val))
		}
	}

	return data
}

// Keys returns an iterator over all header keys in request.
func (h *HeaderCtx) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for k := range h.ctx.Request().Header.All() {
			if !yield(utils.B2S(k)) {
				return
			}
		}
	}
}

// Set sets the response header entries associated with key to the single element value.
// It replaces any existing values associated with key.
func (h *HeaderCtx) Set(key, value string) {
	h.ctx.Response().Header.Set(key, value)
}

// Add adds the key, value pair to the response header. It appends to any existing values associated with key.
func (h *HeaderCtx) Add(key, value string) {
	h.ctx.Response().Header.Add(key, value)
}

// SetAlways sets the response header like Set, and additionally re-applies it if
// the response is later reset to render an error.
// Use it for headers that must remain on error responses, such as CORS headers.
func (h *HeaderCtx) SetAlways(key, value string) {
	h.Set(key, value)
	h.ctx.preserveHeader(key)
}

type headerEntry struct {
	name  string
	value []byte
}

func (c *Context) preserveHeader(name string) {
	for i := range c.alwaysHeaders {
		if c.alwaysHeaders[i].name == name {
			return
		}
	}

	if n := len(c.alwaysHeaders); n < cap(c.alwaysHeaders) {
		c.alwaysHeaders = c.alwaysHeaders[:n+1]
		c.alwaysHeaders[n].name = name

		return
	}

	c.alwaysHeaders = append(c.alwaysHeaders, headerEntry{name: name})
}

func (c *Context) captureAlwaysHeaders() {
	for i := range c.alwaysHeaders {
		c.alwaysHeaders[i].value = append(c.alwaysHeaders[i].value[:0], c.Response().Header.Peek(c.alwaysHeaders[i].name)...)
	}
}

func (c *Context) applyAlwaysHeaders() {
	for i := range c.alwaysHeaders {
		if len(c.alwaysHeaders[i].value) > 0 {
			c.Response().Header.SetBytesV(c.alwaysHeaders[i].name, c.alwaysHeaders[i].value)
		}
	}
}

// Del deletes the values associated with key in both request and response.
func (h *HeaderCtx) Del(key string) {
	h.ctx.Request().Header.Del(key)
	h.ctx.Response().Header.Del(key)
}

// AppendAccessControlExposeHeaders appends the given headers to the Access-Control-Expose-Headers header.
func (h *HeaderCtx) AppendAccessControlExposeHeaders(names ...string) {
	val := h.ctx.Response().Header.Peek(HeaderAccessControlExposeHeaders)
	if len(val) != 0 {
		h.ctx.Response().Header.Set(HeaderAccessControlExposeHeaders, fmt.Sprintf("%s, %s", val, strings.Join(names, ", ")))
	} else {
		h.ctx.Response().Header.Set(HeaderAccessControlExposeHeaders, strings.Join(names, ", "))
	}
}

// InheritAuthorization returns HTTP client request option with inherited authorization from request.
func (h *HeaderCtx) InheritAuthorization() http.RequestOption {
	if auth := h.Get(fasthttp.HeaderAuthorization); auth != "" {
		return http.WithHeader(fasthttp.HeaderAuthorization, auth)
	}

	return nil
}
