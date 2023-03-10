package azugo

import (
	"bytes"
	"strconv"
	"strings"

	"azugo.io/azugo/internal/utils"
)

// QueryCtx represents the key-value pairs in an query string.
type QueryCtx struct {
	noCopy noCopy //nolint:unused,structcheck

	app *App
	ctx *Context
}

// Values returns all values associated with the given key in query.
func (q *QueryCtx) Values(key string) []string {
	data := make([]string, 0, 1)
	q.ctx.Request().URI().QueryArgs().VisitAll(func(k, val []byte) {
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

// String gets the first value associated with the given key in query.
// If there are no values associated with the key or value is empty returns ParamRequiredError error.
func (q *QueryCtx) String(key string) (string, error) {
	v := q.ctx.Request().URI().QueryArgs().Peek(key)
	if len(v) == 0 {
		return "", ParamRequiredError{key}
	}
	return utils.B2S(v), nil
}

// StringOptional gets the first value associated with the given key in query or null if value is empty.
func (q *QueryCtx) StringOptional(key string) *string {
	v := q.ctx.Request().URI().QueryArgs().Peek(key)
	if len(v) == 0 {
		return nil
	}
	s := utils.B2S(v)
	return &s
}

// Int64 returns the value of the parameter as int64.
func (q *QueryCtx) Int64(key string) (int64, error) {
	s, err := q.String(key)
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
func (q *QueryCtx) Int64Optional(key string) (*int64, error) {
	s := q.StringOptional(key)
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
func (q *QueryCtx) Int(key string) (int, error) {
	v, err := q.Int64(key)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}

// IntOptional returns the value of the parameter as optional int or null if value is empty.
func (q *QueryCtx) IntOptional(key string) (*int, error) {
	v, err := q.Int64Optional(key)
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
func (q *QueryCtx) Bool(key string) (bool, error) {
	v, err := q.String(key)
	if err != nil {
		return false, err
	}
	return strings.ToLower(v) == "true" || v == "1", nil
}

// BoolOptional returns the value of the parameter as optional bool or null if value is empty.
//
// Valid values ar "true", "false", "1" and "0".
func (q *QueryCtx) BoolOptional(key string) (*bool, error) {
	v := q.StringOptional(key)
	if v == nil {
		return nil, nil
	}
	iv := strings.ToLower(*v) == "true" || *v == "1"
	return &iv, nil
}
