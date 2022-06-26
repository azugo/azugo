package wsfed

import (
	"bytes"
	"fmt"
	"time"

	"azugo.io/azugo"

	"github.com/beevik/etree"
	xrv "github.com/mattermost/xml-roundtrip-validator"
	dsig "github.com/russellhaering/goxmldsig"
)

const iso8601Layout = "2006-01-02T15:04:05Z"

func parseISO8601Time(s string) (time.Time, error) {
	return time.Parse(iso8601Layout, s)
}

func elementToString(el *etree.Element) (string, error) {
	if el == nil {
		return "", nil
	}
	doc := etree.NewDocument()
	doc.SetRoot(el.Copy())
	return doc.WriteToString()
}

func (p *WsFederation) decodeResponse(resp []byte, opts *tokenParseOptions) (*Token, error) {
	if err := xrv.Validate(bytes.NewReader(resp)); err != nil {
		return nil, err
	}

	doc := etree.NewDocument()
	if err := doc.ReadFromBytes(resp); err != nil {
		return nil, err
	}

	el := doc.Root()
	if el == nil {
		return nil, ErrTokenMalformed
	}

	el = el.FindElement("//Assertion")
	if el == nil {
		return nil, ErrTokenMalformed
	}

	ctx := dsig.NewDefaultValidationContext(p.signCertStore)
	validated, err := ctx.Validate(el)
	if err != nil {
		if err == dsig.ErrMissingSignature {
			return nil, ErrTokenUnverifiable
		} else if err == dsig.ErrInvalidSignature {
			return nil, ErrTokenSignatureInvalid
		}
		return nil, ErrTokenSignatureInvalid
	}

	var raw, signature, validatedRaw string
	if opts.SaveToken {
		var err error
		// Token RAW XML
		raw, err = elementToString(el)
		if err != nil {
			return nil, err
		}

		// Signature XML to token
		signel := el.FindElement("./Signature")
		if signel == nil {
			return nil, ErrTokenMalformed
		}
		signature, err = elementToString(signel)
		if err != nil {
			return nil, err
		}

		// Validated XML token without signature
		validatedRaw, err = elementToString(validated)
		if err != nil {
			return nil, err
		}
	}

	claims := &RegisteredClaims{
		ID:         validated.SelectAttrValue("ID", ""),
		Audience:   make([]string, 0, 1),
		Attributes: make(map[string][]string, 10),
	}

	if iat := validated.SelectAttrValue("IssueInstant", ""); len(iat) > 0 {
		t, err := parseISO8601Time(iat)
		if err != nil {
			return nil, err
		}
		claims.IssuedAt = &t
	}

	if issuer := validated.FindElement("./Issuer"); issuer != nil {
		claims.Issuer = issuer.Text()
	}

	if sub := validated.FindElement("./Subject/NameID"); sub != nil {
		claims.Subject.ID = sub.Text()
		claims.Subject.Format = sub.SelectAttrValue("Format", "")
	}

	if cond := validated.FindElement("./Conditions"); cond != nil {
		if nbf := cond.SelectAttrValue("NotBefore", ""); len(nbf) > 0 {
			t, err := parseISO8601Time(nbf)
			if err != nil {
				return nil, err
			}
			claims.NotBefore = &t
		}
		if exp := cond.SelectAttrValue("NotOnOrAfter", ""); len(exp) > 0 {
			t, err := parseISO8601Time(exp)
			if err != nil {
				return nil, err
			}
			claims.ExpiresAt = &t
		}

		for _, aud := range cond.FindElements("./AudienceRestriction/Audience") {
			claims.Audience = append(claims.Audience, aud.Text())
		}
	}

	for _, attr := range validated.FindElements("./AttributeStatement/Attribute") {
		name := attr.SelectAttrValue("Name", "")
		if len(name) == 0 {
			return nil, ErrTokenMalformed
		}
		vals := make([]string, 0, 1)
		for _, val := range attr.FindElements("./AttributeValue") {
			vals = append(vals, val.Text())
		}
		claims.Attributes[name] = vals
	}

	return &Token{
		Raw:       raw,
		Validated: validatedRaw,
		Signature: signature,
		Claims:    claims,
	}, nil
}

// IsSignoutResponse checks if the request is a signout response.
func (p *WsFederation) IsSignoutResponse(ctx *azugo.Context) bool {
	wa := ctx.Query.StringOptional("wa")
	if wa == nil {
		wa = ctx.Form.StringOptional("wa")
	}

	return wa != nil && (*wa == "wsignoutcleanup1.0" || *wa == "wsignout1.0")
}

// ReadResponse reads the IDP response from the request.
func (p *WsFederation) ReadResponse(ctx *azugo.Context, opt ...TokenParseOption) (*Token, error) {
	if p.IsSignoutResponse(ctx) {
		return nil, nil
	}

	wa, err := ctx.Form.String("wa")
	if err != nil {
		return nil, err
	}

	if wa != "wsignin1.0" {
		return nil, fmt.Errorf("unsupported wa: %s", wa)
	}

	wctx, err := ctx.Form.String("wctx")
	if err != nil {
		return nil, err
	}

	if ok, err := p.NonceStore.Verify(ctx, wctx); !ok || err != nil {
		if !ok {
			return nil, ErrTokenNonceInvalid
		}
		return nil, err
	}

	wresult, err := ctx.Form.String("wresult")
	if err != nil {
		return nil, err
	}

	return p.Parse([]byte(wresult), opt...)
}
