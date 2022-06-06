package wsfed

import (
	"crypto/subtle"
	"time"
)

// Subject holds the unique identifier for the authenticated requestor
type Subject struct {
	ID     string
	Format string
}

// RegisteredClaims are a structured version of the Security Token
type RegisteredClaims struct {
	// Security token issuer
	Issuer string

	// Security token subject
	Subject Subject

	// Audience restrictions
	Audience []string

	// Not on or after restriction
	ExpiresAt *time.Time

	// Not before restriction
	NotBefore *time.Time

	// Issue instant
	IssuedAt *time.Time

	// Assertion ID
	ID string

	// Attribute Statements
	Attributes map[string][]string
}

func verifyAud(aud []string, cmp string, required bool) bool {
	if len(aud) == 0 {
		return !required
	}
	// use a var here to keep constant time compare when looping over a number of claims
	result := false

	var stringClaims string
	for _, a := range aud {
		if subtle.ConstantTimeCompare([]byte(a), []byte(cmp)) != 0 {
			result = true
		}
		stringClaims = stringClaims + a
	}

	// case where "" is sent in one or many aud claims
	if len(stringClaims) == 0 {
		return !required
	}

	return result
}

func verifyExp(exp *time.Time, now time.Time, skew time.Duration, required bool) bool {
	if exp == nil {
		return !required
	}
	now = now.Add(skew)
	return now.Before(*exp)
}

func verifyIat(iat *time.Time, now time.Time, skew time.Duration, required bool) bool {
	if iat == nil {
		return !required
	}
	now = now.Add(-skew)
	return now.After(*iat) || now.Equal(*iat)
}

func verifyNbf(nbf *time.Time, now time.Time, skew time.Duration, required bool) bool {
	if nbf == nil {
		return !required
	}
	now = now.Add(-skew)
	return now.After(*nbf) || now.Equal(*nbf)
}

func verifyIss(iss string, cmp string, required bool) bool {
	if iss == "" {
		return !required
	}
	if subtle.ConstantTimeCompare([]byte(iss), []byte(cmp)) != 0 {
		return true
	} else {
		return false
	}
}
