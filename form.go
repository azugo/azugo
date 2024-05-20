package azugo

import (
	"bytes"
	"mime/multipart"
	"strconv"
	"strings"

	"azugo.io/azugo/internal/utils"

	"github.com/valyala/fasthttp"
)

type formKeyValuer interface {
	Value(key string) string
	Values(key string) []string
	File(key string) *multipart.FileHeader
	Files(key string) []*multipart.FileHeader
	Reset(ctx *Context)
}

// FormCtx represents the post form key-value pairs.
type FormCtx struct {
	noCopy noCopy //nolint:unused,structcheck

	form formKeyValuer

	app *App
	ctx *Context
}

// nilArgs represents noop form key-value pairs.
type nilArgs struct {
	noCopy noCopy //nolint:unused,structcheck
}

func (a *nilArgs) Value(string) string {
	return ""
}

func (a *nilArgs) Values(string) []string {
	return nil
}

func (a *nilArgs) File(string) *multipart.FileHeader {
	return nil
}

func (a *nilArgs) Files(string) []*multipart.FileHeader {
	return nil
}

func (a *nilArgs) Reset(*Context) {}

type postArgs struct {
	noCopy noCopy //nolint:unused,structcheck

	args *fasthttp.Args
}

func (a *postArgs) Value(key string) string {
	return utils.B2S(a.args.Peek(key))
}

func (a *postArgs) Values(key string) []string {
	data := make([]string, 0, 1)

	a.args.VisitAll(func(k, val []byte) {
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

func (a *postArgs) File(string) *multipart.FileHeader {
	return nil
}

func (a *postArgs) Files(string) []*multipart.FileHeader {
	return nil
}

func (a *postArgs) Reset(*Context) {}

type multiPartArgs struct {
	noCopy noCopy //nolint:unused,structcheck

	args *multipart.Form
}

func (a *multiPartArgs) Value(key string) string {
	if v, ok := a.args.Value[key]; ok && len(v) > 0 {
		return v[0]
	}

	return ""
}

func (a *multiPartArgs) Values(key string) []string {
	return a.args.Value[key]
}

func (a *multiPartArgs) File(key string) *multipart.FileHeader {
	if v, ok := a.args.File[key]; ok && len(v) > 0 {
		return a.args.File[key][0]
	}

	return nil
}

func (a *multiPartArgs) Files(key string) []*multipart.FileHeader {
	return a.args.File[key]
}

func (a *multiPartArgs) Reset(ctx *Context) {
	ctx.Context().Request.RemoveMultipartFormFiles()
}

// Values returns all values associated with the given key in query.
func (f *FormCtx) Values(key string) []string {
	return f.form.Values(key)
}

// String gets the first value associated with the given key in form.
// If there are no values associated with the key or value is empty returns ParamRequiredError error.
func (f *FormCtx) String(key string) (string, error) {
	v := f.form.Value(key)
	if len(v) == 0 {
		return "", ParamRequiredError{key}
	}

	return v, nil
}

// StringOptional gets the first value associated with the given key in query or null if value is empty.
func (f *FormCtx) StringOptional(key string) *string {
	v := f.form.Value(key)
	if len(v) == 0 {
		return nil
	}

	return &v
}

// Int64 returns the value of the parameter as int64.
func (f *FormCtx) Int64(key string) (int64, error) {
	s, err := f.String(key)
	if err != nil {
		return 0, err
	}

	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, ParamInvalidError{key, "numeric", err}
	}

	return v, nil
}

// Int64Optional returns the value of the parameter as optional int64 or null if value is empty.
func (f *FormCtx) Int64Optional(key string) (*int64, error) {
	s := f.StringOptional(key)
	if s == nil {
		return nil, nil
	}

	v, err := strconv.ParseInt(*s, 10, 64)
	if err != nil {
		return nil, ParamInvalidError{key, "numeric", err}
	}

	return &v, nil
}

// Int returns the value of the parameter as int.
func (f *FormCtx) Int(key string) (int, error) {
	v, err := f.Int64(key)
	if err != nil {
		return 0, err
	}

	return int(v), nil
}

// IntOptional returns the value of the parameter as optional int or null if value is empty.
func (f *FormCtx) IntOptional(key string) (*int, error) {
	v, err := f.Int64Optional(key)
	if err != nil {
		return nil, err
	}

	if v == nil {
		return nil, nil
	}

	iv := int(*v)

	return &iv, nil
}

// Bool returns the value of the parameter as bool.
//
// Valid values ar "true", "false", "1" and "0".
func (f *FormCtx) Bool(key string) (bool, error) {
	v, err := f.String(key)
	if err != nil {
		return false, err
	}

	return strings.ToLower(v) == "true" || v == "1", nil
}

// BoolOptional returns the value of the parameter as optional bool or null if value is empty.
//
// Valid values ar "true", "false", "1" and "0".
func (f *FormCtx) BoolOptional(key string) (*bool, error) {
	v := f.StringOptional(key)
	if v == nil {
		return nil, nil
	}

	iv := strings.ToLower(*v) == "true" || *v == "1"

	return &iv, nil
}

// File returns uploaded file data.
func (f *FormCtx) File(key string) (*multipart.FileHeader, error) {
	v := f.form.File(key)
	if v == nil {
		return nil, ParamRequiredError{key}
	}

	return v, nil
}

// FileOptional returns uploaded file data if it's provided.
func (f *FormCtx) FileOptional(key string) *multipart.FileHeader {
	return f.form.File(key)
}

// Files returns uploaded files.
func (f *FormCtx) Files(key string) []*multipart.FileHeader {
	return f.form.Files(key)
}
