package wsfed

import (
	"errors"
	"time"
)

// Token parser and validation errors.
var (
	ErrTokenMalformed        = errors.New("token is malformed")
	ErrTokenUnverifiable     = errors.New("token is unverifiable")
	ErrTokenSignatureInvalid = errors.New("token signature is invalid")

	ErrTokenNonceInvalid     = errors.New("token nonce is invalid")
	ErrTokenInvalidAudience  = errors.New("token has invalid audience")
	ErrTokenExpired          = errors.New("token is expired")
	ErrTokenUsedBeforeIssued = errors.New("token used before issued")
	ErrTokenInvalidIssuer    = errors.New("token has invalid issuer")
	ErrTokenNotValidYet      = errors.New("token is not valid yet")
)

// Common claim types.
const (
	ClaimTypeName                      string = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/name"
	ClaimTypeGivenName                 string = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"
	ClaimTypeSurname                   string = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"
	ClaimTypeEmail                     string = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"
	ClaimTypeNameIdentifier            string = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/nameidentifier"
	ClaimTypePrivatePersonalIdentifier string = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/privatepersonalidentifier"
	ClaimTypeSID                       string = "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/sid"
	ClaimTypePrimarySID                string = "http://schemas.microsoft.com/ws/2008/06/identity/claims/primarysid"
	ClaimTypeRole                      string = "http://schemas.microsoft.com/ws/2008/06/identity/claims/role"
	ClaimTypeAction                    string = "http://docs.oasis-open.org/wsfed/authorization/200706/claims/action"
)

// Token represents a WS-Federation token.
type Token struct {
	Raw       string
	Validated string
	Claims    *RegisteredClaims
	Signature string
	Valid     bool
}

// ClaimValue returns the value of the given claim.
func (t *Token) ClaimValue(name string) string {
	c, ok := t.Claims.Attributes[name]
	if ok && len(c) > 0 {
		return c[0]
	}
	return ""
}

// ClaimValues returns the values of the given claim.
func (t *Token) ClaimValues(name string) []string {
	return t.Claims.Attributes[name]
}

type tokenParseOptions struct {
	SaveToken   bool
	ClockSkew   time.Duration
	Audience    string
	CheckExp    bool
	CheckIat    bool
	CheckNbf    bool
	CheckIssuer bool
}

type TokenParseOption interface {
	apply(*tokenParseOptions)
}

// SaveToken is an option to save the token raw and validated XML.
type SaveToken bool

func (o SaveToken) apply(p *tokenParseOptions) {
	p.SaveToken = bool(o)
}

// TokenClockSkew is an option to set the clock skew.
type TokenClockSkew time.Duration

func (o TokenClockSkew) apply(p *tokenParseOptions) {
	p.ClockSkew = time.Duration(o)
}

// TokenValidateIssuer is an option to validate the issuer.
type TokenValidateIssuer bool

func (o TokenValidateIssuer) apply(p *tokenParseOptions) {
	p.CheckIssuer = bool(o)
}

// TokenAudience is an option to set the audience to validate against.
type TokenAudience string

func (o TokenAudience) apply(p *tokenParseOptions) {
	p.Audience = string(o)
}

// TokenValidateIssuedAt is an option to validate issued at time.
type TokenValidateIssuedAt bool

func (o TokenValidateIssuedAt) apply(p *tokenParseOptions) {
	p.CheckIat = bool(o)
}

// TokenValidateExpiresAt is an option to validate expires at time.
type TokenValidateExpiresAt bool

func (o TokenValidateExpiresAt) apply(p *tokenParseOptions) {
	p.CheckExp = bool(o)
}

// TokenValidateNotBefore is an option to validate not before time.
type TokenValidateNotBefore bool

func (o TokenValidateNotBefore) apply(p *tokenParseOptions) {
	p.CheckNbf = bool(o)
}

// Parse parses and validates a WS-Federation token.
func (s *WsFederation) Parse(token []byte, opt ...TokenParseOption) (*Token, error) {
	opts := &tokenParseOptions{
		SaveToken:   false,
		ClockSkew:   s.ClockSkew,
		CheckIssuer: true,
		CheckExp:    true,
		CheckIat:    true,
		CheckNbf:    true,
	}
	for _, o := range opt {
		o.apply(opts)
	}

	t, err := s.decodeResponse(token, opts)
	if err != nil {
		return nil, err
	}

	v := verifyIss(s.Issuer, t.Claims.Issuer, true)
	if opts.CheckIssuer && s.Issuer != "" && !v {
		err = ErrTokenInvalidIssuer
	}

	v = verifyAud(t.Claims.Audience, opts.Audience, true)
	if opts.Audience != "" && !v {
		err = ErrTokenInvalidAudience
	}

	if !verifyIat(t.Claims.IssuedAt, s.clock.Now().UTC(), opts.ClockSkew, true) && opts.CheckIat {
		err = ErrTokenUsedBeforeIssued
	}

	if !verifyExp(t.Claims.ExpiresAt, s.clock.Now().UTC(), opts.ClockSkew, true) && opts.CheckExp {
		err = ErrTokenExpired
	}

	if !verifyNbf(t.Claims.NotBefore, s.clock.Now().UTC(), opts.ClockSkew, true) && opts.CheckNbf {
		err = ErrTokenNotValidYet
	}

	t.Valid = err == nil

	return t, err
}
