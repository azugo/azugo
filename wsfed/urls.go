package wsfed

import (
	"net/url"
	"time"
)

type requestParams struct {
	Wreply string
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

// SigninURL returns the signin URL.
func (p *WsFederation) SigninURL(realm string, options ...RequestOption) (string, error) {
	if err := p.check(p.defaultHttpClient(), false); err != nil {
		return "", err
	}

	u := *p.IDPEndpoint

	rp := &requestParams{}
	for _, o := range options {
		o.apply(rp)
	}
	wctx, err := p.NonceStore.Create()
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
