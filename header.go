package azugo

import (
	"bytes"
	"fmt"
	"strings"

	"azugo.io/azugo/internal/utils"
)

const (
	HeaderAccept                     string = "Accept"
	HeaderTotalCount                 string = "X-Total-Count"
	HeaderLink                       string = "Link"
	HeaderAccessControlExposeHeaders string = "Access-Control-Expose-Headers"
	HeaderContentType                string = "Content-Type"
	HeaderContentDisposition         string = "Content-Disposition"
	HeaderContentTransferEncoding    string = "Content-Transfer-Encoding"
)

const (
	ContentTypeJSON        string = "application/json"
	ContentTypeXML         string = "application/xml"
	ContentTypeOctetStream string = "application/octet-stream"
)

// HeaderCtx represents the key-value pairs in an HTTP header.
type HeaderCtx struct {
	noCopy noCopy //nolint:unused,structcheck

	app *App
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

	h.ctx.Request().Header.VisitAll(func(k, val []byte) {
		if !strings.EqualFold(key, utils.B2S(k)) {
			return
		}

		if bytes.Contains(val, []byte{','}) {
			values := bytes.Split(val, []byte{','})
			for i := 0; i < len(values); i++ {
				data = append(data, utils.B2S(values[i]))
			}
		} else {
			data = append(data, utils.B2S(val))
		}
	})

	return data
}

// Keys returns all header keys in request.
func (h *HeaderCtx) Keys() []string {
	keys := make([]string, 0, h.ctx.Request().Header.Len())

	h.ctx.Request().Header.VisitAll(func(k, _ []byte) {
		keys = append(keys, utils.B2S(k))
	})

	return keys
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
