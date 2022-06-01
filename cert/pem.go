package cert

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
)

const (
	PEMBlockRSAPrivateKey = "RSA PRIVATE KEY"
	PEMBlockECPrivateKey  = "EC PRIVATE KEY"
	PEMBlockCertificate   = "CERTIFICATE"
)

func publicKey(priv interface{}) interface{} {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &k.PublicKey
	case *ecdsa.PrivateKey:
		return &k.PublicKey
	default:
		return nil
	}
}

func pemBlockForKey(priv interface{}) *pem.Block {
	switch k := priv.(type) {
	case *rsa.PrivateKey:
		return &pem.Block{Type: PEMBlockRSAPrivateKey, Bytes: x509.MarshalPKCS1PrivateKey(k)}
	case *ecdsa.PrivateKey:
		b, err := x509.MarshalECPrivateKey(k)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Unable to marshal ECDSA private key: %v", err)
			os.Exit(2)
		}
		return &pem.Block{Type: PEMBlockECPrivateKey, Bytes: b}
	default:
		return nil
	}
}

// DERBytesToPEMBlocks converts certificate DER bytes and optional private key
// to PEM blocks.
// Returns certificate PEM block and private key PEM block.
func DERBytesToPEMBlocks(der []byte, priv interface{}) ([]byte, []byte) {
	out := &bytes.Buffer{}
	pem.Encode(out, &pem.Block{Type: PEMBlockCertificate, Bytes: der})
	cert := append([]byte{}, out.Bytes()...)

	var key []byte
	if priv != nil {
		out.Reset()
		pem.Encode(out, pemBlockForKey(priv))
		key = append([]byte{}, out.Bytes()...)
	}

	return cert, key
}
