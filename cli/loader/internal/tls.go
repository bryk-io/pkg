package internal

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"

	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/net/http"
	"go.bryk.io/pkg/net/rpc"
)

// TLS settings.
type TLS struct {
	// Enable/Disable tls.
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// Load CAs available on the OS.
	SystemCA bool `json:"system_ca" yaml:"system_ca" mapstructure:"system_ca"`

	// Certificate. Either as base64-encoded PEM value or the path to the file.
	Cert string `json:"cert" yaml:"cert" mapstructure:"cert"`

	// Private key. Either as base64-encoded PEM value or the path to the file.
	Key string `json:"key" yaml:"key" mapstructure:"key"`

	// Additional CA(s). Either as base64-encoded PEM value or the path to the file.
	CustomCA []string `json:"custom_ca" yaml:"custom_ca" mapstructure:"custom_ca"`
	certPEM  []byte
	keyPEM   []byte
	caPEM    [][]byte
}

// Validate the provided TLS settings.
func (ct *TLS) Validate() error {
	if !ct.Enabled {
		return nil
	}

	var err error
	// Load cert
	ct.certPEM, err = loadPEM(ct.Cert)
	if err != nil {
		return errors.Wrap(err, "can't load certificate")
	}

	// Load key
	ct.keyPEM, err = loadPEM(ct.Key)
	if err != nil {
		return errors.Wrap(err, "can't load private key")
	}

	// Validate
	if !isKeyPairPEM(ct.certPEM, ct.keyPEM) {
		return errors.New("invalid certificate/key pair")
	}

	// Load custom CAs
	if len(ct.CustomCA) == 0 {
		return nil
	}
	for _, ca := range ct.CustomCA {
		caPEM, err := loadPEM(ca)
		if err != nil {
			return errors.Wrap(err, "invalid custom CA")
		}
		ct.caPEM = append(ct.caPEM, caPEM)
	}
	return nil
}

// Params available when using the loader with a CLI application.
func (ct *TLS) Params(prefix string) []cli.Param {
	return []cli.Param{
		{
			Name:      withPrefix(prefix, "tls", "-"),
			Usage:     "Enable TLS settings",
			FlagKey:   withPrefix(prefix, "tls.enabled", "."),
			ByDefault: false,
		},
		{
			Name:      withPrefix(prefix, "tls-system-ca", "-"),
			Usage:     "Include CAs available on the OS",
			FlagKey:   withPrefix(prefix, "tls.system_ca", "."),
			ByDefault: false,
		},
		{
			Name:      withPrefix(prefix, "tls-cert", "-"),
			Usage:     "TLS certificate",
			FlagKey:   withPrefix(prefix, "tls.cert", "."),
			ByDefault: "",
		},
		{
			Name:      withPrefix(prefix, "tls-key", "-"),
			Usage:     "TLS private key",
			FlagKey:   withPrefix(prefix, "tls.key", "."),
			ByDefault: "",
		},
		{
			Name:      withPrefix(prefix, "tls-custom-ca", "-"),
			Usage:     "Custom certificate authority",
			FlagKey:   withPrefix(prefix, "tls.custom_ca", "."),
			ByDefault: []string{},
		},
	}
}

// Expand the TLS settings and return them on the proper type as specified
// by `ti`.
func (ct *TLS) Expand(ti string) interface{} {
	switch ti {
	case "http":
		return http.TLS{
			Cert:             ct.certPEM,
			PrivateKey:       ct.keyPEM,
			CustomCAs:        ct.caPEM,
			IncludeSystemCAs: ct.SystemCA,
		}
	case "rpc":
		return rpc.WithTLS(rpc.ServerTLSConfig{
			IncludeSystemCAs: ct.SystemCA,
			Cert:             ct.certPEM,
			PrivateKey:       ct.keyPEM,
			CustomCAs:        ct.caPEM,
		})
	default:
		return nil
	}
}

// AuthPEM returns the custom CA certificates used for authentication.
func (ct *TLS) AuthPEM() [][]byte {
	return ct.caPEM
}

// Load a PEM value; either encoded as b64 or from the local filesystem.
func loadPEM(value string) ([]byte, error) {
	// Base64 string
	c, err := base64.StdEncoding.DecodeString(value)
	if err == nil {
		return c, nil
	}

	// Load file
	return ioutil.ReadFile(path.Clean(value))
}

// Validates a certificate/private key pair from it's PEM-encoded byte arrays.
func isKeyPairPEM(cert, key []byte) bool {
	_, err := tls.X509KeyPair(cert, key)
	return err == nil
}

// Join `val` and `prefix` with `sep` in between.
func withPrefix(prefix, val, sep string) string {
	if prefix == "" {
		return val
	}
	return fmt.Sprintf("%s%s%s", prefix, sep, val)
}
