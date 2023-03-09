package azugo

import (
	"errors"
	"fmt"
	"reflect"

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

// ResponseStatusCode is an interface that error can implement to return
// status code that will be set for the response.
type ResponseStatusCode interface {
	StatusCode() int
}

// ErrorResponseError is an error response error details.
type ErrorResponseError struct {
	Type    string `json:"type" xml:"Type"`
	Message string `json:"message" xml:"Message"`
}

// ErrorResponse represents an error response.
type ErrorResponse struct {
	Errors []*ErrorResponseError `json:"errors" xml:"Errors>Error"`
}

func fromSafeError(err SafeError) *ErrorResponseError {
	msg := err.SafeError()
	if len(msg) == 0 {
		return nil
	}

	t := reflect.TypeOf(err)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	return &ErrorResponseError{
		Type:    t.Name(),
		Message: msg,
	}
}

// NewErrorResponse creates an error response from the given error.
func NewErrorResponse(err error) *ErrorResponse {
	if err == nil {
		return nil
	}

	errs := make([]*ErrorResponseError, 0, 1)

	// Detect validation errors
	var verr validator.ValidationErrors
	if errors.As(err, &verr) {
		for _, e := range verr {
			errs = append(errs, &ErrorResponseError{
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

	return &ErrorResponse{
		Errors: errs,
	}
}

// ErrParamRequired is an error that occurs when a required parameter is not provided.
type ErrParamRequired struct {
	Name string
}

func (e ErrParamRequired) Error() string {
	return "parameter required"
}

func (e ErrParamRequired) SafeError() string {
	return fmt.Sprintf(fieldErrMsg, e.Name, e.Name, "required")
}

func (e ErrParamRequired) StatusCode() int {
	return fasthttp.StatusBadRequest
}

// ErrParamInvalid is an error that occurs when a parameter is invalid.
type ErrParamInvalid struct {
	Name string
	Tag  string
	Err  error
}

func (e ErrParamInvalid) Error() string {
	if e.Err == nil {
		return "invalid parameter value"
	}
	return e.Err.Error()
}

func (e ErrParamInvalid) SafeError() string {
	return fmt.Sprintf(fieldErrMsg, e.Name, e.Name, e.Tag)
}

func (e ErrParamInvalid) StatusCode() int {
	return fasthttp.StatusBadRequest
}

// ErrParamInvalid is an error that occurs when user access is denied.
type ErrForbidden struct{}

func (e ErrForbidden) Error() string {
	return "access forbidden"
}

func (e ErrForbidden) StatusCode() int {
	return fasthttp.StatusForbidden
}

// ErrNotFound is an error that occurs when searched resource is not found.
type ErrNotFound struct {
	MissingResource string
}

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("%s not found", e.MissingResource)
}

func (e ErrNotFound) StatusCode() int {
	return fasthttp.StatusNotFound
}
