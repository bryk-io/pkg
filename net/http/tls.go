package http

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/pkg/errors"
	"go.bryk.io/pkg/net/rpc"
)

// TLS defines available settings when enabling secure TLS communications.
type TLS rpc.ServerTLSConfig

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
		ciphers = rpc.RecommendedCiphers
	}
	curves := t.PreferredCurves
	if len(curves) == 0 {
		curves = rpc.RecommendedCurves
	}

	// Base TLS configuration
	return &tls.Config{
		Certificates:             []tls.Certificate{cert},
		CipherSuites:             ciphers,
		CurvePreferences:         curves,
		RootCAs:                  cp,
		PreferServerCipherSuites: true,
		MinVersion:               tls.VersionTLS12,
	}, nil
}
