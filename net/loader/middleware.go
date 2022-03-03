package loader

import (
	"go.bryk.io/pkg/cli"
)

type confMiddleware struct {
	CORS     *confMiddlewareCors     `json:"cors,omitempty" yaml:"cors,omitempty" mapstructure:"cors"`
	HSTS     *confMiddlewareHSTS     `json:"hsts,omitempty" yaml:"hsts,omitempty" mapstructure:"hsts"`
	Metadata *confMiddlewareMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty" mapstructure:"metadata"`
}

func (c *confMiddleware) setDefaults() {
	c.CORS = &confMiddlewareCors{}
	c.HSTS = &confMiddlewareHSTS{}
	c.Metadata = &confMiddlewareMetadata{}
	c.CORS.setDefaults()
	c.HSTS.setDefaults()
	c.Metadata.setDefaults()
}

func (c *confMiddleware) validate() error {
	if err := c.Metadata.validate(); err != nil {
		return err
	}
	return c.HSTS.validate()
}

func (c *confMiddleware) params(segment ...string) []cli.Param {
	var list []cli.Param
	for _, s := range segment {
		switch s {
		case SegmentMiddlewareCORS:
			list = append(list, c.CORS.params()...)
		case SegmentMiddlewareHSTS:
			list = append(list, c.HSTS.params()...)
		case SegmentMiddlewareMetadata:
			list = append(list, c.Metadata.params()...)
		}
	}
	return list
}
