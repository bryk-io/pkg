package rpc

import (
	"encoding/json"

	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/cli/loader/internal"
	"go.bryk.io/pkg/errors"
	"go.bryk.io/pkg/net/rpc"
	rpcWS "go.bryk.io/pkg/net/rpc/ws"
)

// New loader component instance with default values.
func New() *Server {
	return &Server{
		InputValidation:  true,
		NetworkInterface: "all",
		Port:             9999,
		Limits: &rpc.ResourceLimits{
			Connections: 1000,
			Requests:    50,
			Rate:        5000,
		},
		TLS: &internal.TLS{
			Enabled: false,
		},
		WSProxy: &internal.WSProxy{
			Enabled: false,
		},
	}
}

// Server provides a configuration loader module for `rpc.Server` instances.
type Server struct {
	// Perform automatic validation for proto messages that support it.
	InputValidation bool `json:"input_validation" yaml:"input_validation" mapstructure:"input_validation"`

	// Name of the network interface to be used by the server.
	NetworkInterface string `json:"network_interface" yaml:"network_interface" mapstructure:"network_interface"`

	// TPC port for RPC communications.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// UNIX socket for RPC communications.
	UnixSocket string `json:"unix_socket" yaml:"unix_socket" mapstructure:"unix_socket"`

	// Enables certificate-based authentication on the server. If enabled, clients
	// will need to authenticate by providing a valid certificate issued by one of
	// the CAs added on the TLS channel as `CustomCA`. This option is only applicable
	// when operating the server through a TLS channel, otherwise will be ignored.
	AuthByCert bool `json:"auth_by_cert" yaml:"auth_by_cert" mapstructure:"auth_by_cert"`

	// Apply resource limits for RPC server.
	Limits *rpc.ResourceLimits `json:"limits,omitempty" mapstructure:"limits"`

	// TLS settings for secure communications.
	TLS *internal.TLS `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`

	// WebSocket proxy.
	WSProxy *internal.WSProxy `json:"wsproxy,omitempty" yaml:"wsproxy,omitempty" mapstructure:"wsproxy"`
}

// Validate the provided server settings.
func (c *Server) Validate() error {
	if c.UnixSocket != "" && c.Port != 0 {
		return errors.New("can't use unix socket and port simultaneously")
	}
	if c.TLS != nil && c.TLS.Enabled {
		if err := c.TLS.Validate(); err != nil {
			return err
		}
	}
	if c.WSProxy != nil && c.WSProxy.Enabled {
		if err := c.WSProxy.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Params available when using the loader with a CLI application.
func (c *Server) Params() []cli.Param {
	list := []cli.Param{
		{
			Name:      "rpc-port",
			Usage:     "TCP port",
			FlagKey:   "rpc.port",
			ByDefault: c.Port,
		},
		{
			Name:      "rpc-unix-socket",
			Usage:     "Use a UNIX socket as main access point",
			FlagKey:   "rpc.unix_socket",
			ByDefault: c.UnixSocket,
		},
		{
			Name:      "rpc-network-interface",
			Usage:     "Network interface to use to listen for incoming requests",
			FlagKey:   "rpc.network_interface",
			ByDefault: c.NetworkInterface,
		},
		{
			Name:      "rpc-input-validation",
			Usage:     "Automatically detect any errors on received messages",
			FlagKey:   "rpc.input_validation",
			ByDefault: false,
		},
		{
			Name:      "rpc-auth-by-cert",
			Usage:     "Use x509 certificates for client authentication",
			FlagKey:   "rpc.auth_by_cert",
			ByDefault: false,
		},
	}
	if c.Limits != nil {
		list = append(list, []cli.Param{
			{
				Name:      "rpc-limits-connections",
				Usage:     "Maximum number of simultaneous connections",
				FlagKey:   "rpc.limits.connections",
				ByDefault: 1000,
			},
			{
				Name:      "rpc-limits-requests",
				Usage:     "Maximum number of simultaneous requests per-client",
				FlagKey:   "rpc.limits.requests",
				ByDefault: 50,
			},
			{
				Name:      "rpc-limits-rate",
				Usage:     "Maximum number of total requests per-second",
				FlagKey:   "rpc.limits.rate",
				ByDefault: 5000,
			},
		}...)
	}
	if c.TLS != nil {
		list = append(list, c.TLS.Params("rpc")...)
	}
	if c.WSProxy != nil {
		list = append(list, c.WSProxy.Params("rpc")...)
	}
	return list
}

// Expand the server settings and return them as `[]rpc.ServerOption`.
func (c *Server) Expand() interface{} {
	var list []rpc.ServerOption
	list = append(list, rpc.WithPanicRecovery())
	list = append(list, rpc.WithNetworkInterface(c.NetworkInterface))
	if c.InputValidation {
		list = append(list, rpc.WithInputValidation())
	}
	if c.Port != 0 {
		list = append(list, rpc.WithPort(c.Port))
	}
	if c.UnixSocket != "" {
		list = append(list, rpc.WithUnixSocket(c.UnixSocket))
	}
	if c.Limits != nil {
		list = append(list, rpc.WithResourceLimits(*c.Limits))
	}
	if c.TLS != nil && c.TLS.Enabled {
		// nolint:forcetypeassert
		list = append(list, c.TLS.Expand("rpc").(rpc.ServerOption))
		if c.AuthByCert {
			for _, ca := range c.TLS.AuthPEM() {
				list = append(list, rpc.WithAuthByCertificate(ca))
			}
		}
	}
	if c.WSProxy != nil && c.WSProxy.Enabled {
		opts, _ := c.WSProxy.Expand("rpc").([]rpcWS.ProxyOption)
		list = append(list, rpc.WithWebSocketProxy(opts...))
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
