package loader

import (
	"go.bryk.io/pkg/cli"
	xlog "go.bryk.io/pkg/log"
	"go.bryk.io/pkg/otel"
)

// nolint: lll
type confObservability struct {
	ServiceName    string                 `json:"service_name" yaml:"service_name" mapstructure:"service_name"`
	ServiceVersion string                 `json:"service_version" yaml:"service_version" mapstructure:"service_version"`
	Attributes     map[string]interface{} `json:"attributes" yaml:"attributes" mapstructure:"attributes"`
	LogJSON        bool                   `json:"log_json" yaml:"log_json" mapstructure:"log_json"`
}

func (c *confObservability) setDefaults() {}

func (c *confObservability) validate() error {
	return nil
}

func (c *confObservability) params() []cli.Param {
	return []cli.Param{
		{
			Name:      "observability-tracer-name",
			Usage:     "Tracer name",
			FlagKey:   "observability.tracer_name",
			ByDefault: "",
		},
		{
			Name:      "observability-service-name",
			Usage:     "Service name",
			FlagKey:   "observability.service_name",
			ByDefault: "",
		},
		{
			Name:      "observability-service-version",
			Usage:     "Service version",
			FlagKey:   "observability.service_version",
			ByDefault: "",
		},
		{
			Name:      "observability-log-json",
			Usage:     "Produce structured (JSON) log messages",
			FlagKey:   "observability.log_json",
			ByDefault: false,
		},
	}
}

func (c *confObservability) expand() []otel.OperatorOption {
	var opt []otel.OperatorOption
	opt = append(opt, otel.WithLogger(xlog.WithZero(xlog.ZeroOptions{
		PrettyPrint: !c.LogJSON,
		ErrorField:  "error.message",
	})))
	if c.ServiceName != "" {
		opt = append(opt, otel.WithServiceName(c.ServiceName))
	}
	if c.ServiceVersion != "" {
		opt = append(opt, otel.WithServiceVersion(c.ServiceVersion))
	}
	if len(c.Attributes) > 0 {
		opt = append(opt, otel.WithResourceAttributes(c.Attributes))
	}
	return opt
}
