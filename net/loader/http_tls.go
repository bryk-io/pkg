package loader

import (
	"github.com/pkg/errors"
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/http"
)

type confHTTPTLS struct {
	Enabled  bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	SystemCA bool     `json:"system_ca" yaml:"system_ca" mapstructure:"system_ca"`
	Cert     string   `json:"cert" yaml:"cert" mapstructure:"cert"`
	Key      string   `json:"key" yaml:"key" mapstructure:"key"`
	CustomCA []string `json:"custom_ca" yaml:"custom_ca" mapstructure:"custom_ca"`
	certPEM  []byte
	keyPEM   []byte
	caPEM    [][]byte
}

func (ct *confHTTPTLS) validate() error {
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

func (ct *confHTTPTLS) params() []cli.Param {
	return []cli.Param{
		{
			Name:      "http-tls",
			Usage:     "Enable TLS settings",
			FlagKey:   "http.tls.enabled",
			ByDefault: false,
		},
		{
			Name:      "http-tls-system-ca",
			Usage:     "Include CAs available on the OS",
			FlagKey:   "http.tls.system_ca",
			ByDefault: false,
		},
		{
			Name:      "http-tls-cert",
			Usage:     "TLS certificate",
			FlagKey:   "http.tls.cert",
			ByDefault: "",
		},
		{
			Name:      "http-tls-key",
			Usage:     "TLS private key",
			FlagKey:   "http.tls.key",
			ByDefault: "",
		},
		{
			Name:      "http-tls-custom-ca",
			Usage:     "Custom certificate authority",
			FlagKey:   "http.tls.custom_ca",
			ByDefault: []string{},
		},
	}
}

func (ct *confHTTPTLS) expand() http.TLS {
	return http.TLS{
		Cert:             ct.certPEM,
		PrivateKey:       ct.keyPEM,
		CustomCAs:        ct.caPEM,
		IncludeSystemCAs: ct.SystemCA,
	}
}
