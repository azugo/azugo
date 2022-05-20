package azugo

import (
	"reflect"
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

func NewErrorResponse(err error) *ErrorResponse {
	if err == nil {
		return nil
	}

	serr, ok := err.(SafeError)
	if !ok {
		return nil
	}

	errs := make([]*ErrorResponseError, 0, 1)
	if r := fromSafeError(serr); r != nil {
		errs = append(errs, r)
	}

	if len(errs) == 0 {
		return nil
	}

	return &ErrorResponse{
		Errors: errs,
	}
}
