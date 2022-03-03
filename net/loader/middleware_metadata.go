package loader

import (
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/middleware"
)

type confMiddlewareMetadata struct {
	middleware.ContextMetadataOptions `yaml:",inline" mapstructure:",squash"`
}

func (c *confMiddlewareMetadata) setDefaults() {
	c.Headers = []string{}
}

func (c *confMiddlewareMetadata) validate() error {
	return nil
}

func (c *confMiddlewareMetadata) params() []cli.Param {
	return []cli.Param{
		{
			Name:      "metadata-headers",
			Usage:     "Headers (in non-canonical form) to preserve as context metadata",
			FlagKey:   "middleware.metadata.headers",
			ByDefault: []string{},
		},
	}
}

func (c *confMiddlewareMetadata) expand() middleware.ContextMetadataOptions {
	return c.ContextMetadataOptions
}
