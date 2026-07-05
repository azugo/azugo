package azugo

import (
	"fmt"
	"iter"
	"strings"

	"azugo.io/azugo/internal/utils"

	"azugo.io/core/http"
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

	for k, val := range h.All() {
		if !strings.EqualFold(key, k) {
			continue
		}

		if strings.Contains(val, ",") {
			data = append(data, strings.Split(val, ",")...)
		} else {
			data = append(data, val)
		}
	}

	return data
}

// All returns an iterator over all request header entries.
func (h *HeaderCtx) All() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for k, val := range h.ctx.Request().Header.All() {
			if !yield(utils.B2S(k), utils.B2S(val)) {
				return
			}
		}
	}
}

// AllInOrder returns an iterator over all request header entries in the order they were received.
//
// It is slightly slower than All because it has to reparse the raw headers to get the order.
func (h *HeaderCtx) AllInOrder() iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for k, val := range h.ctx.Request().Header.AllInOrder() {
			if !yield(utils.B2S(k), utils.B2S(val)) {
				return
			}
		}
	}
}

// AcceptsEncoding returns true if the request accepts the given content encoding.
func (h *HeaderCtx) AcceptsEncoding(encoding string) bool {
	return h.ctx.Request().Header.HasAcceptEncoding(encoding)
}

// Keys returns an iterator over all header keys in request.
func (h *HeaderCtx) Keys() iter.Seq[string] {
	return func(yield func(string) bool) {
		for k := range h.All() {
			if !yield(k) {
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
	val := h.ctx.Response().Header.Peek(http.HeaderAccessControlExposeHeaders)
	if len(val) != 0 {
		h.ctx.Response().Header.Set(http.HeaderAccessControlExposeHeaders, fmt.Sprintf("%s, %s", val, strings.Join(names, ", ")))
	} else {
		h.ctx.Response().Header.Set(http.HeaderAccessControlExposeHeaders, strings.Join(names, ", "))
	}
}

// InheritAuthorization returns HTTP client request option with inherited authorization from request.
func (h *HeaderCtx) InheritAuthorization() http.RequestOption {
	if auth := h.Get(http.HeaderAuthorization); auth != "" {
		return http.WithHeader(http.HeaderAuthorization, auth)
	}

	return nil
}
