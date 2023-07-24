package otel

import (
	"encoding/json"

	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/log"
	"go.bryk.io/pkg/otel/sdk"
)

// New loader component instance with default values.
func New() *Operator {
	return &Operator{
		ServiceName:    "my-service",
		ServiceVersion: "0.1.0",
	}
}

// Operator provides a configuration loader module for `otel.Operator` instances.
type Operator struct {
	// Service identifier.
	ServiceName string `json:"service_name" yaml:"service_name" mapstructure:"service_name"`

	// Service version tag.
	ServiceVersion string `json:"service_version" yaml:"service_version" mapstructure:"service_version"`

	// Additional attributes for the service.
	Attributes map[string]interface{} `json:"attributes" yaml:"attributes" mapstructure:"attributes"`

	// Produce structured logging messages.
	LogJSON bool `json:"log_json" yaml:"log_json" mapstructure:"log_json"`

	// Capture host metrics.
	HostMetrics bool `json:"host_metrics" yaml:"host_metrics" mapstructure:"host_metrics"`

	// Capture Go runtime metrics.
	RuntimeMetrics bool `json:"runtime_metrics" yaml:"runtime_metrics" mapstructure:"runtime_metrics"`
}

// Validate the provided operator settings.
func (c *Operator) Validate() error {
	return nil
}

// Params available when using the loader with a CLI application.
func (c *Operator) Params() []cli.Param {
	return []cli.Param{
		{
			Name:      "otel-service-name",
			Usage:     "Service name",
			FlagKey:   "otel.service_name",
			ByDefault: c.ServiceName,
		},
		{
			Name:      "otel-service-version",
			Usage:     "Service version",
			FlagKey:   "otel.service_version",
			ByDefault: c.ServiceVersion,
		},
		{
			Name:      "otel-host-metrics",
			Usage:     "Capture host metrics",
			FlagKey:   "otel.host_metrics",
			ByDefault: false,
		},
		{
			Name:      "otel-runtime-metrics",
			Usage:     "Capture Go runtime metrics",
			FlagKey:   "otel.runtime_metrics",
			ByDefault: false,
		},
	}
}

// Expand operator settings and return them as a `[]sdk.Option`.
func (c *Operator) Expand() interface{} {
	var opt []sdk.Option
	opt = append(opt, sdk.WithBaseLogger(log.WithZero(log.ZeroOptions{
		PrettyPrint: !c.LogJSON,
		ErrorField:  "error.message",
	})))
	if c.ServiceName != "" {
		opt = append(opt, sdk.WithServiceName(c.ServiceName))
	}
	if c.ServiceVersion != "" {
		opt = append(opt, sdk.WithServiceVersion(c.ServiceVersion))
	}
	if len(c.Attributes) > 0 {
		opt = append(opt, sdk.WithResourceAttributes(c.Attributes))
	}
	if c.HostMetrics {
		opt = append(opt, sdk.WithHostMetrics())
	}
	if c.RuntimeMetrics {
		opt = append(opt, sdk.WithRuntimeMetrics(0))
	}
	return opt
}

// Restore operator settings from the provided data structure.
func (c *Operator) Restore(data map[string]interface{}) error {
	// use intermediary data structure
	restore, _ := json.Marshal(data)
	if err := json.Unmarshal(restore, c); err != nil {
		return err
	}
	return c.Validate()
}
