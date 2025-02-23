package rpc

import (
	"crypto/tls"
	"crypto/x509"

	"go.bryk.io/pkg/errors"
)

// HTTP/2 over TLS uses the "h2" protocol identifier.
// https://datatracker.ietf.org/doc/html/rfc7540#section-3.3
const alpnProtocolIdentifier = "h2"

// RecommendedCiphers provides a default list of secure/modern ciphers.
var RecommendedCiphers = []uint16{
	tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
	tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
	tls.TLS_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305_SHA256,
	tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305_SHA256,
}

// RecommendedCurves provides a sane list of curves with assembly implementations
// for performance and constant time to protect against side-channel attacks.
var RecommendedCurves = []tls.CurveID{
	tls.CurveP521,
	tls.CurveP384,
	tls.CurveP256,
	tls.X25519,
	// tls.X25519MLKEM768,
}

// ServerTLSConfig provides available settings to enable secure TLS communications.
type ServerTLSConfig struct {
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

// Generate a proper TLS configuration to use on the server.
func serverTLSConf(opts ServerTLSConfig) (*tls.Config, error) {
	// Load key/pair
	cert, err := tls.X509KeyPair(opts.Cert, opts.PrivateKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed to load key pair")
	}

	// Prepare cert pool
	var cp *x509.CertPool
	if opts.IncludeSystemCAs {
		cp, err = x509.SystemCertPool()
		if err != nil {
			return nil, errors.Wrap(err, "failed to load system CAs")
		}
	} else {
		cp = x509.NewCertPool()
	}

	// Append custom CA certs
	if len(opts.CustomCAs) > 0 {
		for _, c := range opts.CustomCAs {
			if !cp.AppendCertsFromPEM(c) {
				return nil, errors.New("failed to append provided CA certificates")
			}
		}
	}

	// Setup ciphers and curves
	if opts.SupportedCiphers == nil {
		opts.SupportedCiphers = RecommendedCiphers
	}
	if opts.PreferredCurves == nil {
		opts.PreferredCurves = RecommendedCurves
	}

	// Base TLS configuration
	conf := &tls.Config{
		Certificates:     []tls.Certificate{cert},
		CipherSuites:     opts.SupportedCiphers,
		CurvePreferences: opts.PreferredCurves,
		RootCAs:          cp,
		MinVersion:       tls.VersionTLS12,
		NextProtos:       []string{alpnProtocolIdentifier},
	}
	return conf, nil
}
