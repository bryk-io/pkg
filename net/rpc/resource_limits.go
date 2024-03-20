package rpc

// ResourceLimits allows setting constrains for the RPC server.
type ResourceLimits struct {
	// Maximum number of simultaneous RPC connections (clients).
	Connections uint32 `json:"connections" yaml:"connections" mapstructure:"connections"`

	// Maximum number of simultaneous RPC calls per-client.
	Requests uint32 `json:"requests" yaml:"requests" mapstructure:"requests"`

	// Maximum number of RPC calls per-second (total).
	Rate uint32 `json:"rate" yaml:"rate" mapstructure:"rate"`
}
