package wsfed

import (
	"context"
	"net/url"
	"time"
)

type requestParams struct {
	Wreply string
	Params []*customRequestParam
}

type customRequestParam struct {
	Name  string
	Value string
}

// RequestOption is an optional parameters for the request.
type RequestOption interface {
	apply(p *requestParams)
}

// WithRequestWreply is an optional reply URL parameter for request.
type WithRequestWreply string

func (o WithRequestWreply) apply(p *requestParams) {
	p.Wreply = string(o)
}

func (o *customRequestParam) apply(p *requestParams) {
	p.Params = append(p.Params, o)
}

// WithRequestParam is an optional custom parameter for request.
func WithRequestParam(name, value string) RequestOption {
	return &customRequestParam{
		Name:  name,
		Value: value,
	}
}

// SigninURL returns the signin URL.
func (p *WsFederation) SigninURL(ctx context.Context, realm string, options ...RequestOption) (string, error) {
	if err := p.check(false); err != nil {
		return "", err
	}

	u := *p.IDPEndpoint

	rp := &requestParams{
		Params: make([]*customRequestParam, 0),
	}
	for _, o := range options {
		o.apply(rp)
	}

	wctx, err := p.NonceStore.Create(ctx)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Add("wa", "wsignin1.0")
	params.Add("wtrealm", realm)

	if rp.Wreply != "" {
		params.Add("wreply", rp.Wreply)
	}

	params.Add("wct", p.clock.Now().UTC().Format(time.RFC3339))
	params.Add("wctx", wctx)

	for _, param := range rp.Params {
		params.Add(param.Name, param.Value)
	}

	u.RawQuery = params.Encode()

	return u.String(), nil
}

// SignoutURL returns the signout URL.
func (p *WsFederation) SignoutURL(realm string, options ...RequestOption) (string, error) {
	if err := p.check(false); err != nil {
		return "", err
	}

	u := *p.IDPEndpoint

	rp := &requestParams{}
	for _, o := range options {
		o.apply(rp)
	}

	params := url.Values{}
	params.Add("wa", "wsignout1.0")
	params.Add("wtrealm", realm)

	if rp.Wreply != "" {
		params.Add("wreply", rp.Wreply)
	}

	u.RawQuery = params.Encode()

	return u.String(), nil
}
