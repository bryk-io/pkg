package rpc

import (
	"crypto/tls"
	"crypto/x509"

	"go.bryk.io/x/errors"
)

// ClientTLSConfig defines the configuration options available when establishing
// a secure communication channel with a server.
type ClientTLSConfig struct {
	// Whether to include system CAs.
	IncludeSystemCAs bool

	// Custom certificate authorities to include when accepting TLS connections.
	CustomCAs [][]byte
}

// Generate a proper TLS configuration to use on the client side.
func clientTLSConf(opts ClientTLSConfig) (*tls.Config, error) {
	conf := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Prepare cert pool
	var err error
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

	conf.RootCAs = cp
	return conf, nil
}
