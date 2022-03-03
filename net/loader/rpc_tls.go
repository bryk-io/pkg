package loader

import (
	"github.com/pkg/errors"
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/rpc"
)

type confTLS struct {
	Enabled    bool     `json:"enabled" yaml:"enabled" mapstructure:"enabled"`
	SystemCA   bool     `json:"system_ca" yaml:"system_ca" mapstructure:"system_ca"`
	Cert       string   `json:"cert" yaml:"cert" mapstructure:"cert"`
	Key        string   `json:"key" yaml:"key" mapstructure:"key"`
	CustomCA   []string `json:"custom_ca" yaml:"custom_ca" mapstructure:"custom_ca"`
	AuthByCert []string `json:"auth_by_certificate" yaml:"auth_by_certificate" mapstructure:"auth_by_certificate"`

	certPEM []byte
	keyPEM  []byte
	caPEM   [][]byte
	authPEM [][]byte
}

func (ct *confTLS) validate() error {
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

	// Auth CA
	for _, ca := range ct.AuthByCert {
		caPEM, err := loadPEM(ca)
		if err != nil {
			return errors.Wrap(err, "invalid CA for authentication")
		}
		ct.authPEM = append(ct.authPEM, caPEM)
	}
	return nil
}

func (ct *confTLS) params() []cli.Param {
	return []cli.Param{
		{
			Name:      "rpc-tls",
			Usage:     "Enable TLS settings",
			FlagKey:   "rpc.tls.enabled",
			ByDefault: false,
		},
		{
			Name:      "rpc-tls-system-ca",
			Usage:     "Include CAs available on the OS",
			FlagKey:   "rpc.tls.system_ca",
			ByDefault: false,
		},
		{
			Name:      "rpc-tls-cert",
			Usage:     "TLS certificate",
			FlagKey:   "rpc.tls.cert",
			ByDefault: "",
		},
		{
			Name:      "rpc-tls-key",
			Usage:     "TLS private key",
			FlagKey:   "rpc.tls.key",
			ByDefault: "",
		},
		{
			Name:      "rpc-tls-custom-ca",
			Usage:     "Custom certificate authority",
			FlagKey:   "rpc.tls.custom_ca",
			ByDefault: []string{},
		},
		{
			Name:      "rpc-tls-auth-by-certificate",
			Usage:     "Authenticate clients by-certificate from the specified CA",
			FlagKey:   "rpc.tls.auth_by_certificate",
			ByDefault: []string{},
		},
	}
}

func (ct *confTLS) expand() []rpc.ServerOption {
	if !ct.Enabled {
		return []rpc.ServerOption{}
	}
	list := make([]rpc.ServerOption, len(ct.authPEM)+1)
	list = append(list, rpc.WithTLS(rpc.ServerTLSConfig{
		IncludeSystemCAs: ct.SystemCA,
		Cert:             ct.certPEM,
		PrivateKey:       ct.keyPEM,
		CustomCAs:        ct.caPEM,
	}))
	for _, ca := range ct.authPEM {
		list = append(list, rpc.WithAuthByCertificate(ca))
	}
	return list
}
