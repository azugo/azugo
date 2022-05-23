package azugo

import (
	"github.com/goccy/go-json"
)

// Body represents the request body.
type Body struct {
	noCopy noCopy //nolint:unused,structcheck

	app *App
	ctx *Context
}

// Bytes returns the request body as raw bytes.
func (b *Body) Bytes() []byte {
	return b.ctx.Request().Body()
}

// JSON unmarshals the request body into provided structure.
// Optionally calls Validate method of the structure if it
// implements validation.Validator interface.
func (b *Body) JSON(v interface{}) error {
	buf := b.Bytes()
	if len(buf) == 0 {
		return ErrParamRequired{"body"}
	}
	if err := json.Unmarshal(buf, v); err != nil {
		return ErrParamInvalid{"body", "json", err}
	}
	if v, ok := v.(Validator); ok {
		return v.Validate(b.ctx)
	}
	return nil
}
