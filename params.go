package azugo

import (
	"strconv"
)

// ParamsCtx represents the parameters of route URL.
type ParamsCtx struct {
	noCopy noCopy //nolint:unused,structcheck

	app *App
	ctx *Context
}

// Has the given name specified in route params.
func (p *ParamsCtx) Has(key string) bool {
	return p.ctx.context.Value(key) != nil
}

// String gets the first value associated with the given name in route params.
func (p *ParamsCtx) String(key string) string {
	v := p.ctx.context.Value(key)
	if v == nil {
		return ""
	}
	return v.(string)
}

// Int64 returns the value of the parameter as int64.
func (p *ParamsCtx) Int64(key string) (int64, error) {
	s := p.String(key)
	if s == "" {
		return 0, nil
	}
	v, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, ErrParamInvalid{key, "numeric", err}
	}
	return v, nil
}

// Int returns the value of the parameter as int.
func (p *ParamsCtx) Int(key string) (int, error) {
	v, err := p.Int64(key)
	if err != nil {
		return 0, err
	}
	return int(v), nil
}
