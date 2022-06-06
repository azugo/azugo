package wsfed

import (
	"encoding/xml"
	"time"

	"github.com/beevik/etree"
)

// EntityDescriptor is a WS-Federation entity descriptor.
type EntityDescriptor struct {
	XMLName       xml.Name   `xml:"urn:oasis:names:tc:SAML:2.0:metadata EntityDescriptor"`
	ID            *string    `xml:",attr,omitempty"`
	ValidUntil    *time.Time `xml:"validUntil,attr,omitempty"`
	CacheDuration *time.Time `xml:"cacheDuration,attr,omitempty"`
	Signature     *etree.Element
	// SAML 2.0 8.3.6 Entity Identifier could be used to represent issuer
	EntityID       string            `xml:"entityID,attr"`
	Extensions     *Extensions       `xml:"Extensions,omitempty"`
	RoleDescriptor []*RoleDescriptor `xml:"RoleDescriptor,omitempty"`
}

// DigestMethod is a digest type specification
type DigestMethod struct {
	Algorithm string `xml:",attr,omitempty"`
}

// SigningMethod is a signing type specification
type SigningMethod struct {
	Algorithm  string `xml:",attr"`
	MinKeySize string `xml:"MinKeySize,attr,omitempty"`
	MaxKeySize string `xml:"MaxKeySize,attr,omitempty"`
}

// Extensions is a list of extensions
type Extensions struct {
	DigestMethod  *DigestMethod  `xml:",omitempty"`
	SigningMethod *SigningMethod `xml:",omitempty"`
}

// RoleDescriptor is a role descriptor
type RoleDescriptor struct {
	ID                       string        `xml:",attr,omitempty"`
	Type                     string        `xml:"http://www.w3.org/2001/XMLSchema-instance type,attr"`
	ValidUntil               *time.Time    `xml:"validUntil,attr,omitempty"`
	CacheDuration            time.Duration `xml:"cacheDuration,attr,omitempty"`
	Signature                *etree.Element
	KeyDescriptors           []*KeyDescriptor          `xml:"KeyDescriptor,omitempty"`
	PassiveRequestorEndpoint *PassiveRequestorEndpoint `xml:"PassiveRequestorEndpoint,omitempty"`
}

// KeyDescriptor represents the XMLSEC object of the same name
type KeyDescriptor struct {
	Use               string              `xml:"use,attr"`
	KeyInfo           *KeyInfo            `xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
	EncryptionMethods []*EncryptionMethod `xml:"EncryptionMethod"`
}

// EncryptionMethod represents the XMLSEC object of the same name
type EncryptionMethod struct {
	Algorithm string `xml:"Algorithm,attr"`
}

// KeyInfo represents the XMLSEC object of the same name
type KeyInfo struct {
	XMLName  xml.Name  `xml:"http://www.w3.org/2000/09/xmldsig# KeyInfo"`
	X509Data *X509Data `xml:"X509Data"`
}

// X509Data represents the XMLSEC object of the same name
type X509Data struct {
	XMLName          xml.Name           `xml:"http://www.w3.org/2000/09/xmldsig# X509Data"`
	X509Certificates []*X509Certificate `xml:"X509Certificate"`
}

// X509Certificate represents the XMLSEC object of the same name
type X509Certificate struct {
	XMLName xml.Name `xml:"http://www.w3.org/2000/09/xmldsig# X509Certificate"`
	Data    string   `xml:",chardata"`
}

// PassiveRequestorEndpoint represents WS Federation Passive Requestor Endpoint
type PassiveRequestorEndpoint struct {
	XMLName           xml.Name           `xml:"http://docs.oasis-open.org/wsfed/federation/200706 PassiveRequestorEndpoint"`
	EndpointReference *EndpointReference `xml:"EndpointReference"`
}

// EndpointReference represents WSA addressing endpoint reference
type EndpointReference struct {
	XMLName xml.Name `xml:"http://www.w3.org/2005/08/addressing EndpointReference"`
	Address string   `xml:"http://www.w3.org/2005/08/addressing Address"`
}
