package http

import (
	"crypto/tls"
	"crypto/x509"

	"go.bryk.io/pkg/errors"
)

// TLS defines available settings when enabling secure TLS communications.
type TLS struct {
	// Server certificate, PEM-encoded.
	Cert []byte

	// Server private key, PEM-encoded.
	PrivateKey []byte

	// List of ciphers to allow.
	SupportedCiphers []uint16

	// Server preferred curves configuration.
	PreferredCurves []tls.CurveID

	// Whether to include system CAs.
	IncludeSystemCAs bool

	// Custom certificate authorities to include when accepting TLS connections.
	CustomCAs [][]byte
}

// RecommendedCiphers provides a default list of secure/modern ciphers.
var recommendedCiphers = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
}

// RecommendedCurves provides a sane list of curves with assembly implementations
// for performance and constant time to protect against side-channel attacks.
var recommendedCurves = []tls.CurveID{
	tls.CurveP521,
	tls.CurveP384,
	tls.CurveP256,
	tls.X25519,
}

// Expand returns a TLS configuration instance based on the provided
// settings.
func (t TLS) Expand() (*tls.Config, error) {
	// Load key/pair
	cert, err := tls.X509KeyPair(t.Cert, t.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load key pair")
	}

	// Prepare cert pool
	var cp *x509.CertPool
	if t.IncludeSystemCAs {
		cp, err = x509.SystemCertPool()
		if err != nil {
			return nil, errors.Wrap(err, "failed to load system CAs")
		}
	} else {
		cp = x509.NewCertPool()
	}

	// Append custom CA certs
	if len(t.CustomCAs) > 0 {
		for _, c := range t.CustomCAs {
			if !cp.AppendCertsFromPEM(c) {
				return nil, errors.New("failed to append provided CA certificates")
			}
		}
	}

	// Setup ciphers and curves
	ciphers := t.SupportedCiphers
	if len(ciphers) == 0 {
		ciphers = recommendedCiphers
	}
	curves := t.PreferredCurves
	if len(curves) == 0 {
		curves = recommendedCurves
	}

	// Base TLS configuration
	return &tls.Config{
		Certificates:     []tls.Certificate{cert},
		CipherSuites:     ciphers,
		CurvePreferences: curves,
		RootCAs:          cp,
		MinVersion:       tls.VersionTLS12,
	}, nil
}
