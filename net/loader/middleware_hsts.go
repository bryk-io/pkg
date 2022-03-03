package loader

import (
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/middleware"
)

type confMiddlewareHSTS struct {
	middleware.HSTSOptions `yaml:",inline" mapstructure:",squash"`
}

func (c *confMiddlewareHSTS) setDefaults() {
	c.HSTSOptions = middleware.DefaultHSTSOptions()
}

func (c *confMiddlewareHSTS) validate() error {
	return nil
}

func (c *confMiddlewareHSTS) params() []cli.Param {
	return []cli.Param{
		{
			Name:      "hsts-max-age",
			Usage:     "Duration (in hours) that the HSTS is valid for",
			FlagKey:   "middleware.hsts.max_age",
			ByDefault: 24 * 365,
		},
		{
			Name:      "hsts-host-override",
			Usage:     "Override redirection URL",
			FlagKey:   "middleware.hsts.host_override",
			ByDefault: "",
		},
		{
			Name:      "hsts-accept-forwarded-proto",
			Usage:     "Accept the X-Forwarded-Proto header as proof of SSL",
			FlagKey:   "middleware.hsts.accept_forwarded_proto",
			ByDefault: false,
		},
		{
			Name:      "hsts-send-preload-directive",
			Usage:     "Sets whether the preload directive should be set",
			FlagKey:   "middleware.hsts.send_preload_directive",
			ByDefault: false,
		},
		{
			Name:      "hsts-include-subdomains",
			Usage:     "Apply the HSTS policy to subdomains",
			FlagKey:   "middleware.hsts.include_subdomains",
			ByDefault: false,
		},
	}
}

func (c *confMiddlewareHSTS) expand() middleware.HSTSOptions {
	return c.HSTSOptions
}
