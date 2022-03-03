package loader

import (
	"strings"

	"github.com/pkg/errors"
	"go.bryk.io/pkg/cli"
	"go.bryk.io/pkg/net/http"
	"go.bryk.io/pkg/net/middleware"
	"go.bryk.io/pkg/net/rpc"
	"go.bryk.io/pkg/net/rpc/ws"
	"go.bryk.io/pkg/otel"
	"gopkg.in/yaml.v2"
)

const (
	// SegmentRPC enable RPC server settings.
	SegmentRPC = "rpc"

	// SegmentHTTP enable HTTP server settings.
	SegmentHTTP = "http"

	// SegmentWebsocket enable Websocket Proxy settings.
	SegmentWebsocket = "websocket"

	// SegmentObservability enable observability operator settings.
	SegmentObservability = "observability"

	// SegmentMiddlewareCORS enable CORS HTTP middleware settings.
	SegmentMiddlewareCORS = "middleware.cors"

	// SegmentMiddlewareHSTS enable HSTS HTTP middleware settings.
	SegmentMiddlewareHSTS = "middleware.hsts"

	// SegmentMiddlewareMetadata enable Metadata HTTP middleware settings.
	SegmentMiddlewareMetadata = "middleware.metadata"
)

// Data holds all configuration parameters available on for
// a helper instance.
type Data struct {
	// RPC server settings.
	RPC *confRPC `json:"rpc" yaml:"rpc" mapstructure:"rpc"`

	// HTTP server settings.
	HTTP *confHTTP `json:"http" yaml:"http" mapstructure:"http"`

	// HTTP middleware.
	Middleware *confMiddleware `json:"middleware" yaml:"middleware" mapstructure:"middleware"`

	// WebSocket proxy settings.
	Websocket *confWS `json:"websocket" yaml:"websocket" mapstructure:"websocket"`

	// Observability operator settings.
	Observability *confObservability `json:"observability" yaml:"observability" mapstructure:"observability"`
}

// Helper instance can be used to simplify configuration management
// of complex network services.
type Helper struct {
	// Configuration parameters handled by the helper instance.
	Data *Data
}

// New will set up a new helper instance with default settings.
func New() *Helper {
	c := &Helper{
		Data: &Data{
			RPC:           &confRPC{},
			HTTP:          &confHTTP{},
			Websocket:     &confWS{},
			Middleware:    &confMiddleware{},
			Observability: &confObservability{},
		},
	}
	c.Data.RPC.setDefaults()
	c.Data.HTTP.setDefaults()
	c.Data.Websocket.setDefaults()
	c.Data.Middleware.setDefaults()
	c.Data.Observability.setDefaults()
	return c
}

// FromYAML restore a helper instance from YAML-encoded settings.
func FromYAML(content []byte) (*Helper, error) {
	d := &Data{}
	if err := yaml.Unmarshal(content, d); err != nil {
		return nil, errors.Wrap(err, "YAML decode error")
	}
	h := &Helper{Data: d}
	if err := h.Validate(); err != nil {
		return nil, err
	}
	return h, nil
}

// Validate the configuration parameters set.
func (h *Helper) Validate() error {
	var err error
	if err = h.Data.RPC.validate(); err != nil {
		return err
	}
	if err = h.Data.HTTP.validate(); err != nil {
		return err
	}
	if err = h.Data.Middleware.validate(); err != nil {
		return err
	}
	if err = h.Data.Websocket.validate(); err != nil {
		return err
	}
	return h.Data.Observability.validate()
}

// Params return CLI definitions for the specified segment(s).
func (h *Helper) Params(segments ...string) []cli.Param {
	var list []cli.Param
	for _, s := range segments {
		switch {
		case s == SegmentRPC:
			list = append(list, h.Data.RPC.params()...)
		case s == SegmentHTTP:
			list = append(list, h.Data.HTTP.params()...)
		case s == SegmentWebsocket:
			list = append(list, h.Data.Websocket.params()...)
		case s == SegmentObservability:
			list = append(list, h.Data.Observability.params()...)
		case strings.HasPrefix(s, "middleware."):
			list = append(list, h.Data.Middleware.params(s)...)
		}
	}
	return list
}

// ToYAML return current configuration settings in YAML format.
func (h *Helper) ToYAML() ([]byte, error) {
	return yaml.Marshal(h.Data)
}

// MiddlewareCORS configuration options.
func (h *Helper) MiddlewareCORS() middleware.CORSOptions {
	return h.Data.Middleware.CORS.expand()
}

// MiddlewareHSTS configuration options.
func (h *Helper) MiddlewareHSTS() middleware.HSTSOptions {
	return h.Data.Middleware.HSTS.expand()
}

// MiddlewareMetadata configuration options.
func (h *Helper) MiddlewareMetadata() middleware.ContextMetadataOptions {
	return h.Data.Middleware.Metadata.expand()
}

// Observability operator configuration options.
func (h *Helper) Observability() []otel.OperatorOption {
	return h.Data.Observability.expand()
}

// Websocket proxy configuration options.
func (h *Helper) Websocket() []ws.ProxyOption {
	return h.Data.Websocket.expand()
}

// ServerRPC configuration options.
func (h *Helper) ServerRPC() []rpc.ServerOption {
	return h.Data.RPC.expand()
}

// HTTPGateway configuration options.
func (h *Helper) HTTPGateway() []rpc.HTTPGatewayOption {
	if !h.Data.RPC.HTTPGateway.Enabled {
		return nil
	}
	return []rpc.HTTPGatewayOption{
		rpc.WithGatewayPort(h.Data.RPC.HTTPGateway.Port),
	}
}

// HTTPServer configuration options.
func (h *Helper) HTTPServer() []http.Option {
	return h.Data.HTTP.expand()
}
