package azugo

import (
	"errors"
	"fmt"
	"reflect"

	"azugo.io/core/http"
	"github.com/go-playground/validator/v10"
	"github.com/valyala/fasthttp"
)

const (
	fieldErrMsg = "Key: '%s' Error:Field validation for '%s' failed on the '%s' tag"
)

// SafeError is an interface that error can implement to return message
// that can be safely returned to the client.
type SafeError interface {
	SafeError() string
}

func fromSafeError(err SafeError) *http.ErrorResponseError {
	msg := err.SafeError()
	if len(msg) == 0 {
		return nil
	}

	t := reflect.TypeOf(err)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return &http.ErrorResponseError{
		Type:    t.Name(),
		Message: msg,
	}
}

// NewErrorResponse creates an error response from the given error.
func NewErrorResponse(err error) *http.ErrorResponse {
	if err == nil {
		return nil
	}

	errs := make([]*http.ErrorResponseError, 0, 1)

	// Detect validation errors
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		for _, e := range verr {
			errs = append(errs, &http.ErrorResponseError{
				Type:    "FieldError",
				Message: e.Error(),
			})
		}
	}

	// Detect safe error
	if serr, ok := err.(SafeError); ok {
		if r := fromSafeError(serr); r != nil {
			errs = append(errs, r)
		}
	}

	if len(errs) == 0 {
		return nil
	}

	return &http.ErrorResponse{
		Errors: errs,
	}
}

// ParamRequiredError is an error that occurs when a required parameter is not provided.
type ParamRequiredError struct {
	Name string
}

func (e ParamRequiredError) Error() string {
	return "parameter required"
}

func (e ParamRequiredError) SafeError() string {
	return fmt.Sprintf(fieldErrMsg, e.Name, e.Name, "required")
}

func (ParamRequiredError) StatusCode() int {
	return fasthttp.StatusBadRequest
}

// ParamInvalidError is an error that occurs when a parameter is invalid.
type ParamInvalidError struct {
	Name string
	Tag  string
	Err  error
}

func (e ParamInvalidError) Error() string {
	if e.Err == nil {
		return "invalid parameter value"
	}

	return e.Err.Error()
}

func (e ParamInvalidError) SafeError() string {
	return fmt.Sprintf(fieldErrMsg, e.Name, e.Name, e.Tag)
}

func (ParamInvalidError) StatusCode() int {
	return fasthttp.StatusBadRequest
}

// BadRequestError is an error that occurs when request is malformed.
type BadRequestError struct {
	Description string
}

func (e BadRequestError) Error() string {
	if e.Description == "" {
		return "malformed request"
	}

	return "malformed request: " + e.Description
}

func (BadRequestError) StatusCode() int {
	return fasthttp.StatusBadRequest
}
