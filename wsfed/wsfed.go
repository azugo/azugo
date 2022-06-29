package wsfed

import (
	"crypto/x509"
	"net/url"
	"sync"
	"time"

	"azugo.io/azugo"
	"azugo.io/azugo/cache"
	"azugo.io/azugo/token/nonce"

	"github.com/jonboulle/clockwork"
	dsig "github.com/russellhaering/goxmldsig"
)

const (
	// Cache key to store nonce in cache.
	WsFederationNonceCacheKey string = "wsfed-nonce"
)

// WsFederation is a WS-Federation service to communicate with IDP.
type WsFederation struct {
	// MetadataURL is the URL to the WS-Federation metadata.
	MetadataURL *url.URL
	// InsecureSkipVerify skips the verification of the IDP HTTPS certificate.
	InsecureSkipVerify bool
	// IDPEndpoint is the URL to the IDP endpoint for passive authentication.
	IDPEndpoint *url.URL
	// Issuer of the token
	Issuer string
	// ClockSkew is the maximum allowed clock skew.
	ClockSkew time.Duration
	// NonceStore is the nonce store.
	NonceStore nonce.Store

	lock          sync.RWMutex
	signCertStore dsig.X509CertificateStore
	ready         bool
	clock         clockwork.Clock
}

// New creates a new WS-Federation service instance.
func New(app *azugo.App, metadataURL string) (*WsFederation, error) {
	var u *url.URL
	if metadataURL != "" {
		var err error
		u, err = url.Parse(metadataURL)
		if err != nil {
			return nil, err
		}
	}

	st, err := cache.Create[bool](app.Cache(), WsFederationNonceCacheKey, cache.DefaultTTL(10*time.Minute))
	if err != nil {
		return nil, err
	}

	return &WsFederation{
		MetadataURL: u,
		ClockSkew:   5 * time.Minute,
		NonceStore:  nonce.NewCacheNonceStore(st),

		clock: clockwork.NewRealClock(),
		signCertStore: &dsig.MemoryX509CertificateStore{
			Roots: []*x509.Certificate{},
		},
	}, nil
}

// ClearCertificateStore clears the certificate store.
func (p *WsFederation) ClearCertificateStore() {
	p.signCertStore = &dsig.MemoryX509CertificateStore{
		Roots: []*x509.Certificate{},
	}
}

// AddTrustedSigningCertificate adds a trusted certificate to the certificate store.
func (p *WsFederation) AddTrustedSigningCertificate(cert *x509.Certificate) {
	s := p.signCertStore.(*dsig.MemoryX509CertificateStore)
	s.Roots = append(s.Roots, cert)
}

// RefreshMetadata updates the metadata.
func (p *WsFederation) RefreshMetadata() error {
	if p.MetadataURL == nil {
		return nil
	}

	return p.check(p.defaultHttpClient(), true)
}

// Ready returns true if the service is ready.
func (p *WsFederation) Ready() bool {
	if p.check(p.defaultHttpClient(), false) != nil {
		return false
	}

	p.lock.RLock()
	defer p.lock.RUnlock()
	return p.ready
}
