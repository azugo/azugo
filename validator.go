package azugo

import (
	"azugo.io/core/validation"
)

// Validator is an interface that can be implemented by structs
// that can be called to validate the struct.
type Validator interface {
	// Validate validates the struct and returns validation error.
	Validate(ctx *Context) error
}

// Validate returns validation service instance.
func (ctx *Context) Validate() *validation.Validate {
	return ctx.app.Validate()
}
