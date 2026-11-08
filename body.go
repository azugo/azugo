package azugo

import (
	"bytes"
	"mime"
	"mime/multipart"

	"encoding/xml"
	"io"

	"azugo.io/azugo/internal/utils"

	"github.com/goccy/go-json"
)

// BodyCtx represents the request body.
type BodyCtx struct {
	noCopy noCopy

	ctx *Context
}

// ReadAll returns the request body as raw bytes enforcing the body size limit.
func (b *BodyCtx) ReadAll() ([]byte, error) {
	return b.ctx.readBody()
}

// Bytes returns the request body as raw bytes or nil if it can't be read.
func (b *BodyCtx) Bytes() []byte {
	buf, _ := b.ctx.readBody()

	return buf
}

// WriteTo copies the request raw body to the provided writer.
//
// Warning: Always returns 0 as not possible to determine
// the number of bytes written.
func (b *BodyCtx) WriteTo(w io.Writer) (int64, error) {
	return 0, b.ctx.context.Request.BodyWriteTo(w)
}

// Reader returns the request body as a stream limited by the body size limit.
func (b *BodyCtx) Reader() (io.Reader, error) {
	if !b.ctx.Request().IsBodyStream() {
		buf, err := b.ctx.readBody()
		if err != nil {
			return nil, err
		}

		return bytes.NewReader(buf), nil
	}

	return b.ctx.limitedBodyStream()
}

// MultipartReader returns a streaming multipart reader for the request body.
func (b *BodyCtx) MultipartReader() (*multipart.Reader, error) {
	_, params, err := mime.ParseMediaType(utils.B2S(b.ctx.Request().Header.ContentType()))
	if err != nil || params["boundary"] == "" {
		return nil, BadRequestError{"invalid multipart content type", err}
	}

	r, err := b.Reader()
	if err != nil {
		return nil, err
	}

	return multipart.NewReader(r, params["boundary"]), nil
}

// JSON unmarshals the request body into provided structure.
// Optionally calls Validate method of the structure if it
// implements validation.Validator interface.
func (b *BodyCtx) JSON(v any) error {
	buf, err := b.ctx.readBody()
	if err != nil {
		return err
	}

	if len(buf) == 0 {
		return ParamRequiredError{"body"}
	}

	if err = json.Unmarshal(buf, v); err != nil {
		return BadRequestError{"invalid content", err}
	}

	if v, ok := v.(Validator); ok {
		return v.Validate(b.ctx)
	}

	return nil
}

// XML unmarshals the request body into provided structure.
func (b *BodyCtx) XML(v any) error {
	buf, err := b.ctx.readBody()
	if err != nil {
		return err
	}

	if len(buf) == 0 {
		return ParamRequiredError{"body"}
	}

	if err = xml.Unmarshal(buf, v); err != nil {
		return BadRequestError{"invalid content", err}
	}

	return nil
}
