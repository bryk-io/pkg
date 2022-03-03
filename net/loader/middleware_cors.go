package loader

import (
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/middleware"
)

type confMiddlewareCors struct {
	middleware.CORSOptions `yaml:",inline" mapstructure:",squash"`
}

func (c *confMiddlewareCors) setDefaults() {
	c.OptionsStatusCode = 200
	c.AllowCredentials = true
	c.IgnoreOptions = false
	c.AllowedHeaders = []string{"content-type"}
	c.MaxAge = 300
}

func (c *confMiddlewareCors) params() []cli.Param {
	return []cli.Param{
		{
			Name:      "cors-allow-credentials",
			Usage:     "User-agent may pass authentication details with the request",
			FlagKey:   "middleware.cors.allow_credentials",
			ByDefault: false,
		},
		{
			Name:      "cors-ignore-options",
			Usage:     "Ignore OPTIONS requests passing them through to the next handler",
			FlagKey:   "middleware.cors.ignore_options",
			ByDefault: false,
		},
		{
			Name:      "cors-allowed-headers",
			Usage:     "List of allowed headers in a CORS request",
			FlagKey:   "middleware.cors.allowed_headers",
			ByDefault: []string{},
		},
		{
			Name:      "cors-allowed-methods",
			Usage:     "Allowed methods in the Access-Control-Allow-Methods header",
			FlagKey:   "middleware.cors.allowed_methods",
			ByDefault: []string{},
		},
		{
			Name:      "cors-allowed-origins",
			Usage:     "Sets the allowed origins for CORS requests",
			FlagKey:   "middleware.cors.allowed_origins",
			ByDefault: []string{},
		},
		{
			Name:      "cors-exposed-headers",
			Usage:     "Headers that will not be stripped out by the user-agent",
			FlagKey:   "middleware.cors.exposed_headers",
			ByDefault: []string{},
		},
		{
			Name:      "cors-max-age",
			Usage:     "Maximum age (in seconds) between preflight requests",
			FlagKey:   "middleware.cors.max_age",
			ByDefault: 300,
		},
		{
			Name:      "cors-options-status-code",
			Usage:     "Status code returned for OPTIONS requests",
			FlagKey:   "middleware.cors.options_status_code",
			ByDefault: 200,
		},
	}
}

func (c *confMiddlewareCors) expand() middleware.CORSOptions {
	return c.CORSOptions
}
