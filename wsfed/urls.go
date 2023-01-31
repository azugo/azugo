package wsfed

import (
	"context"
	"net/url"
	"time"
)

type requestParams struct {
	Wreply   string
	Language string
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

// WithRequestLang is an optional language URL parameter for request (ISO 639-1 format).
type WithRequestLang string

func (o WithRequestLang) apply(p *requestParams) {
	p.Language = string(o)
}

// SigninURL returns the signin URL.
func (p *WsFederation) SigninURL(ctx context.Context, realm string, options ...RequestOption) (string, error) {
	if err := p.check(p.defaultHttpClient(), false); err != nil {
		return "", err
	}

	u := *p.IDPEndpoint

	rp := &requestParams{}
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
	if rp.Language != "" {
		params.Add("lang", rp.Language)
	}

	u.RawQuery = params.Encode()

	return u.String(), nil
}

// SignoutURL returns the signout URL.
func (p *WsFederation) SignoutURL(realm string, options ...RequestOption) (string, error) {
	if err := p.check(p.defaultHttpClient(), false); err != nil {
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
