package wsfed

import "errors"

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

// Token represents a WS-Federation token.
type Token struct {
	Raw       string
	Claims    *RegisteredClaims
	Signature string
	Valid     bool
}

// Parse parses and validates a WS-Federation token.
func (s *WsFederation) Parse(token []byte, aud string) (*Token, error) {
	t, err := s.decodeResponse(token)
	if err != nil {
		return nil, err
	}

	v := verifyIss(s.Issuer, t.Claims.Issuer, true)
	if s.Issuer != "" && !v {
		err = ErrTokenInvalidIssuer
	}

	v = verifyAud(t.Claims.Audience, aud, true)
	if aud != "" && !v {
		err = ErrTokenInvalidAudience
	}

	if verifyIat(t.Claims.IssuedAt, s.clock.Now(), s.ClockSkew, true) {
		err = ErrTokenUsedBeforeIssued
	}

	if verifyExp(t.Claims.ExpiresAt, s.clock.Now(), s.ClockSkew, true) {
		err = ErrTokenExpired
	}

	if verifyNbf(t.Claims.NotBefore, s.clock.Now(), s.ClockSkew, true) {
		err = ErrTokenNotValidYet
	}

	t.Valid = err == nil

	return t, err
}
