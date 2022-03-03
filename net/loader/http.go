package loader

import (
	"time"

	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/http"
)

type confHTTP struct {
	// TPC port for HTTP communications.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// Keep-alive timeout.
	IdleTimeout int `json:"idle_timeout" yaml:"idle_timeout" mapstructure:"idle_timeout"`

	// Secure communication settings.
	TLS *confHTTPTLS `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`
}

func (c *confHTTP) setDefaults() {
	c.Port = 8080
	c.IdleTimeout = 5
	c.TLS = &confHTTPTLS{Enabled: false}
}

func (c *confHTTP) validate() error {
	if c.TLS != nil {
		return c.TLS.validate()
	}
	return nil
}

func (c *confHTTP) params() []cli.Param {
	list := []cli.Param{
		{
			Name:      "http-port",
			Usage:     "TCP port for the HTTP(S) server",
			FlagKey:   "http.port",
			ByDefault: 8080,
		},
		{
			Name:      "http-idle-timeout",
			Usage:     "Maximum time (in seconds) to wait for 'keep-alive' requests",
			FlagKey:   "http.idle_timeout",
			ByDefault: 5,
		},
	}
	return append(list, c.TLS.params()...)
}

func (c *confHTTP) expand() []http.Option {
	var list []http.Option
	list = append(list, http.WithPort(c.Port))
	if c.IdleTimeout > 0 {
		list = append(list, http.WithIdleTimeout(time.Duration(c.IdleTimeout)*time.Second))
	}
	if c.TLS != nil && c.TLS.Enabled {
		list = append(list, http.WithTLS(c.TLS.expand()))
	}
	return list
}
