package cert

import (
	"bytes"
	"encoding/pem"
	"os"
)

// LoadTLSCertificate loads a PEM-encoded certificate and private key from
// the specified file.
func LoadTLSCertificate(path string) ([]byte, []byte, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, err
	}
	var cert, key []byte
	for {
		block, rest := pem.Decode(raw)
		if block == nil {
			break
		}
		if block.Type == PEMBlockCertificate {
			out := &bytes.Buffer{}
			pem.Encode(out, block)
			cert = out.Bytes()
		} else if block.Type == PEMBlockRSAPrivateKey || block.Type == PEMBlockECPrivateKey {
			out := &bytes.Buffer{}
			pem.Encode(out, block)
			key = out.Bytes()
		}
		raw = rest
	}

	return cert, key, nil
}
