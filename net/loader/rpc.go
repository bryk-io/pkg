package loader

import (
	"github.com/pkg/errors"
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/rpc"
)

// nolint: lll
type confRPC struct {
	// Perform automatic validation for proto messages that support it.
	InputValidation bool `json:"input_validation" yaml:"input_validation" mapstructure:"input_validation"`

	// Name of the network interface to be used by the server.
	NetworkInterface string `json:"network_interface" yaml:"network_interface" mapstructure:"network_interface"`

	// TPC port for RPC communications.
	Port int `json:"port" yaml:"port" mapstructure:"port"`

	// UNIX socket for RPC communications.
	UnixSocket string `json:"unix_socket" yaml:"unix_socket" mapstructure:"unix_socket"`

	// HTTP gateway options.
	HTTPGateway *confHTTPGateway `json:"http_gateway" yaml:"http_gateway" mapstructure:"http_gateway"`

	// Apply resource limits for RPC server.
	ResourceLimits *rpc.ResourceLimits `json:"resource_limits,omitempty" yaml:"resource_limits,omitempty" mapstructure:"resource_limits"`

	// TLS settings for secure communications.
	TLS *confTLS `json:"tls,omitempty" yaml:"tls,omitempty" mapstructure:"tls"`
}

type confHTTPGateway struct {
	Enabled bool `json:"enabled" yaml:"enabled" mapstructure:"enabled"`

	// If enabled and the port is set to '0', the same RPC port will be used.
	Port int `json:"port" yaml:"port" mapstructure:"port"`
}

func (c *confRPC) setDefaults() {
	c.InputValidation = true
	c.NetworkInterface = "all"
	c.Port = 9999
	c.UnixSocket = ""
	c.ResourceLimits = &rpc.ResourceLimits{
		Connections: 1000,
		Requests:    50,
		Rate:        5000,
	}
	c.TLS = &confTLS{
		Enabled:  false,
		SystemCA: true,
	}
	c.HTTPGateway = &confHTTPGateway{
		Enabled: false,
		Port:    0,
	}
}

func (c *confRPC) validate() error {
	if c.UnixSocket != "" && c.Port != 0 {
		return errors.New("can't use unix socket and port simultaneously")
	}
	if c.TLS != nil {
		if err := c.TLS.validate(); err != nil {
			return err
		}
	}
	return nil
}

func (c *confRPC) params() []cli.Param {
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
			Name:      "rpc-http-gateway",
			Usage:     "Enable HTTP access to the RPC server",
			FlagKey:   "rpc.http_gateway.enabled",
			ByDefault: false,
		},
		{
			Name:      "rpc-http-gateway-port",
			Usage:     "TCP port for HTTP access, with '0' the same RPC port is used",
			FlagKey:   "rpc.http_gateway.port",
			ByDefault: 0,
		},
	}
	if c.ResourceLimits != nil {
		list = append(list, []cli.Param{
			{
				Name:      "rpc-resource-limits-connections",
				Usage:     "Maximum number of simultaneous connections",
				FlagKey:   "rpc.resource_limits.connections",
				ByDefault: 1000,
			},
			{
				Name:      "rpc-resource-limits-requests",
				Usage:     "Maximum number of simultaneous requests per-client",
				FlagKey:   "rpc.resource_limits.requests",
				ByDefault: 50,
			},
			{
				Name:      "rpc-resource-limits-rate",
				Usage:     "Maximum number of total requests per-second",
				FlagKey:   "rpc.resource_limits.rate",
				ByDefault: 5000,
			},
		}...)
	}
	if c.TLS != nil {
		list = append(list, c.TLS.params()...)
	}
	return list
}

func (c *confRPC) expand() []rpc.ServerOption {
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
	if c.ResourceLimits != nil {
		list = append(list, rpc.WithResourceLimits(*c.ResourceLimits))
	}
	if c.TLS != nil {
		list = append(list, c.TLS.expand()...)
	}
	return list
}
