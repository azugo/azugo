package azugo

import (
	"encoding/xml"
	"io"

	"github.com/goccy/go-json"
)

// BodyCtx represents the request body.
type BodyCtx struct {
	noCopy noCopy //nolint:unused,structcheck

	app *App
	ctx *Context
}

// Bytes returns the request body as raw bytes.
func (b *BodyCtx) Bytes() []byte {
	return b.ctx.Request().Body()
}

// Copy copies the request raw body to the provided writer.
//
// Warning: Always returns 0 as not possible to determine
// the number of bytes written.
func (b *BodyCtx) WriteTo(w io.Writer) (int64, error) {
	return 0, b.ctx.context.Request.BodyWriteTo(w)
}

// JSON unmarshals the request body into provided structure.
// Optionally calls Validate method of the structure if it
// implements validation.Validator interface.
func (b *BodyCtx) JSON(v any) error {
	buf := b.Bytes()
	if len(buf) == 0 {
		return ParamRequiredError{"body"}
	}

	if err := json.Unmarshal(buf, v); err != nil {
		return ParamInvalidError{"body", "json", err}
	}

	if v, ok := v.(Validator); ok {
		return v.Validate(b.ctx)
	}

	return nil
}

// XML unmarshals the request body into provided structure.
func (b *BodyCtx) XML(v any) error {
	buf := b.Bytes()
	if len(buf) == 0 {
		return ParamRequiredError{"body"}
	}

	if err := xml.Unmarshal(buf, v); err != nil {
		return ParamInvalidError{"body", "xml", err}
	}

	return nil
}
