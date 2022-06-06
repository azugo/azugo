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

func elementToString(el *etree.Element) (string, error) {
	if el == nil {
		return "", nil
	}
	doc := etree.NewDocument()
	doc.SetRoot(el.Copy())
	return doc.WriteToString()
}

func (p *WsFederation) decodeResponse(resp []byte) (*Token, error) {
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

	el = validated.FindElement("//Assertion")
	if el == nil {
		return nil, ErrTokenMalformed
	}
	raw, err := elementToString(el)
	if err != nil {
		return nil, err
	}

	signel := el.FindElement("//Signature")
	if signel == nil {
		return nil, ErrTokenMalformed
	}
	signature, err := elementToString(signel)
	if err != nil {
		return nil, err
	}

	claims := &RegisteredClaims{
		ID:         el.SelectAttrValue("ID", ""),
		Audience:   make([]string, 0, 1),
		Attributes: make(map[string][]string, 10),
	}

	if iat := el.SelectAttrValue("IssueInstant", ""); len(iat) > 0 {
		t, err := time.Parse(time.RFC3339, iat)
		if err != nil {
			return nil, err
		}
		claims.IssuedAt = &t
	}

	if issuer := el.FindElement("./Issuer"); issuer != nil {
		claims.Issuer = issuer.Text()
	}

	if sub := el.FindElement("./Subject/NameID"); sub != nil {
		claims.Subject.ID = sub.Text()
		claims.Subject.Format = sub.SelectAttrValue("Format", "")
	}

	if cond := el.FindElement("./Conditions"); cond != nil {
		if nbf := cond.SelectAttrValue("NotBefore", ""); len(nbf) > 0 {
			t, err := time.Parse(time.RFC3339, nbf)
			if err != nil {
				return nil, err
			}
			claims.NotBefore = &t
		}
		if exp := cond.SelectAttrValue("NotOnOrAfter", ""); len(exp) > 0 {
			t, err := time.Parse(time.RFC3339, exp)
			if err != nil {
				return nil, err
			}
			claims.ExpiresAt = &t
		}

		for _, aud := range cond.FindElements("./AudienceRestriction/Audience") {
			claims.Audience = append(claims.Audience, aud.Text())
		}
	}

	for _, attr := range el.FindElements("./AttributeStatement/Attribute") {
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
func (p *WsFederation) ReadResponse(ctx *azugo.Context, aud string) (*Token, error) {
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

	if ok, err := p.NonceStore.Verify(wctx); !ok || err != nil {
		if !ok {
			return nil, ErrTokenNonceInvalid
		}
		return nil, err
	}

	wresult, err := ctx.Form.String("wresult")
	if err != nil {
		return nil, err
	}

	return p.Parse([]byte(wresult), aud)
}
