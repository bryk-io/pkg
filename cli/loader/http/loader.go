package http

import (
	"encoding/json"
	"time"

	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/cli/loader/internal"
	"go.bryk.io/pkg/net/http"
)

// New loader component instance with default values.
func New() *Server {
	return &Server{
		Port:        8080,
		IdleTimeout: 5,
		TLS: &internal.TLS{
			Enabled: false,
		},
	}
}

// Server provides a configuration loader module for `http.Server` instances.
type Server struct {
	// TPC port for HTTP communications.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// Keep-alive timeout.
	IdleTimeout int `json:"idle_timeout" yaml:"idle_timeout" mapstructure:"idle_timeout"`

	// TLS settings.
	TLS *internal.TLS `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`
}

// Validate the provided server settings.
func (c *Server) Validate() error {
	if c.TLS != nil && c.TLS.Enabled {
		return c.TLS.Validate()
	}
	return nil
}

// Params available when using the loader with a CLI application.
func (c *Server) Params() []cli.Param {
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
	if c.TLS != nil {
		list = append(list, c.TLS.Params("http")...)
	}
	return list
}

// Expand the server settings and return them as `[]http.Option`.
func (c *Server) Expand() interface{} {
	var list []http.Option
	list = append(list, http.WithPort(c.Port))
	if c.IdleTimeout > 0 {
		list = append(list, http.WithIdleTimeout(time.Duration(c.IdleTimeout)*time.Second))
	}
	// nolint:forcetypeassert
	if c.TLS != nil && c.TLS.Enabled {
		list = append(list, http.WithTLS(c.TLS.Expand("http").(http.TLS)))
	}
	return list
}

// Restore server settings from the provided data structure.
func (c *Server) Restore(data map[string]interface{}) error {
	restore, _ := json.Marshal(data)
	if err := json.Unmarshal(restore, c); err != nil {
		return err
	}
	return c.Validate()
}
