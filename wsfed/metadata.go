package wsfed

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"azugo.io/azugo/internal/wsfed"

	"github.com/valyala/fasthttp"
)

const (
	// HTTPPostBinding is the official URN for the HTTP-POST binding (transport).
	HTTPPostBinding string = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-POST"

	// HTTPRedirectBinding is the official URN for the HTTP-Redirect binding (transport).
	HTTPRedirectBinding string = "urn:oasis:names:tc:SAML:2.0:bindings:HTTP-Redirect"

	// SecurityTokenServiceType is the official WS-Federation type for the Security Token Service (STS).
	SecurityTokenServiceType string = "SecurityTokenServiceType"

	// KeyDescriptorUseSigning is the official use for a key descriptor that is used for signing.
	KeyDescriptorUseSigning string = "signing"

	// KeyDescriptorUseEncryption is the official use for a key descriptor that is used for encryption.
	KeyDescriptorUseEncryption string = "encryption"
)

func (p *WsFederation) defaultHTTPClient() *fasthttp.Client {
	return &fasthttp.Client{
		NoDefaultUserAgentHeader: true,
		TLSConfig: &tls.Config{
			//nolint:gosec
			InsecureSkipVerify: p.InsecureSkipVerify,
		},
	}
}

func (p *WsFederation) check(client *fasthttp.Client, force bool) error {
	p.lock.RLock()
	if p.ready && !force {
		p.lock.RUnlock()

		return nil
	}
	p.lock.RUnlock()
	p.lock.Lock()
	defer p.lock.Unlock()

	if p.MetadataURL == nil {
		if p.IDPEndpoint == nil {
			return errors.New("no MetadataURL or IDPEndpoint set")
		}

		p.ready = true

		return nil
	}

	req := fasthttp.AcquireRequest()
	req.Header.SetUserAgentBytes(p.ua)
	req.Header.SetMethod(fasthttp.MethodGet)
	req.SetRequestURI(p.MetadataURL.String())

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	err := client.Do(req, resp)
	fasthttp.ReleaseRequest(req)

	if err != nil {
		return fmt.Errorf("failed to connect to the WS-Federation server: %w", err)
	}

	if resp.StatusCode() != fasthttp.StatusOK {
		return fmt.Errorf("WS-Federation server returned unexpected status code (%d) for metadata URL", resp.StatusCode())
	}

	metadata := &wsfed.EntityDescriptor{}
	if err := xml.Unmarshal(resp.Body(), metadata); err != nil {
		return fmt.Errorf("failed to unmarshal WS-Federation server metadata: %w", err)
	}

	p.ClearCertificateStore()

	p.Issuer = metadata.EntityID

	for _, r := range metadata.RoleDescriptor {
		if !strings.HasSuffix(r.Type, ":"+SecurityTokenServiceType) && r.Type != SecurityTokenServiceType {
			continue
		}

		for _, kd := range r.KeyDescriptors {
			if kd.Use != KeyDescriptorUseSigning {
				continue
			}

			for _, cert := range kd.KeyInfo.X509Data.X509Certificates {
				if cert.Data == "" {
					continue
				}

				certData, err := base64.StdEncoding.DecodeString(cert.Data)
				if err != nil {
					return err
				}

				idpCert, err := x509.ParseCertificate(certData)
				if err != nil {
					return err
				}

				p.AddTrustedSigningCertificate(idpCert)
			}
		}

		addr, err := url.Parse(r.PassiveRequestorEndpoint.EndpointReference.Address)
		if err != nil {
			return err
		}

		p.IDPEndpoint = addr
	}

	p.ready = true

	return nil
}
