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

func (e ParamRequiredError) StatusCode() int {
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

func (e ParamInvalidError) StatusCode() int {
	return fasthttp.StatusBadRequest
}

// BadRequestError is an error that occurs when request is malformed.
type BadRequestError struct {
	Description string
}

func (e BadRequestError) Error() string {
	return fmt.Sprintf("malformed request: %s", e.Description)
}

func (e BadRequestError) StatusCode() int {
	return fasthttp.StatusBadRequest
}

// ForbiddenError is an error that occurs when user access is denied.
type ForbiddenError struct{}

func (e ForbiddenError) Error() string {
	return "access forbidden"
}

func (e ForbiddenError) StatusCode() int {
	return fasthttp.StatusForbidden
}

// NotFoundError is an error that occurs when searched resource is not found.
type NotFoundError struct {
	MissingResource string
}

func (e NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", e.MissingResource)
}

func (e NotFoundError) StatusCode() int {
	return fasthttp.StatusNotFound
}
