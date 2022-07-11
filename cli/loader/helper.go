package loader

import (
	"fmt"

	"go.bryk.io/pkg/cli"
	"go.bryk.io/x/errors"
)

// Component elements represent discrete configuration portions of a larger
// application. By segmenting the settings under individual components we
// make the logic and implementation easier to reason about, use and maintain.
type Component interface {
	// Validate all settings and report any errors.
	Validate() error

	// Params returns the required configuration when exposing the component
	// settings as part of a CLI application.
	Params() []cli.Param

	// Expand all component settings and return them on the proper format, i.e.,
	// as intended for consumption (context specific).
	Expand() interface{}

	// Restore settings coming from an external source.
	Restore(data map[string]interface{}) error
}

// Helper instance can be used to simplify configuration management
// of complex services.
type Helper struct {
	comp   map[string]Component
	prefix string
}

// New will set up a new helper instance with default settings.
func New(options ...Option) (*Helper, error) {
	h := &Helper{
		comp:   make(map[string]Component),
		prefix: "",
	}
	for _, opt := range options {
		if err := opt(h); err != nil {
			return nil, err
		}
	}
	return h, nil
}

// Validate the configuration parameters set.
func (h *Helper) Validate() error {
	for _, s := range h.comp {
		if err := s.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// Params return CLI definitions for the specified component(s). If no component(s)
// is specified this method will return CLI param definitions for all components
// registered in the helper instance.
func (h *Helper) Params(comp ...string) []cli.Param {
	var params []cli.Param
	if len(comp) > 0 {
		// Load params for selected components
		for _, c := range comp {
			params = append(params, h.comp[c].Params()...)
		}
	} else {
		// Load params for all components
		for _, s := range h.comp {
			params = append(params, s.Params()...)
		}
	}

	// Apply global prefix to all commands
	if h.prefix != "" {
		for i := range params {
			params[i].FlagKey = fmt.Sprintf("%s.%s", h.prefix, params[i].FlagKey)
		}
	}
	return params
}

// Register a new component element.
func (h *Helper) Register(name string, comp Component) {
	h.comp[name] = comp
}

// Expand returns the specified component settings as intended for consumption
// (context specific). If `name` is not available, or if the component doesn't
// support expanding its settings, this method returns `nil`.
func (h *Helper) Expand(name string) interface{} {
	if s, ok := h.comp[name]; ok {
		return s.Expand()
	}
	return nil
}

// Export settings for all registered components as a portable data structure.
// This is useful, for example, to encode the settings as a JSON or YAML file
// for simpler store and restore operations.
func (h *Helper) Export() interface{} {
	if h.prefix != "" {
		return map[string]interface{}{h.prefix: h.comp}
	}
	return h.comp
}

// Restore previously exported settings.
func (h *Helper) Restore(data map[string]interface{}) error {
	// Unpack prefix from data structure if required
	var src map[string]interface{}
	src = data
	if h.prefix != "" {
		var ok bool
		src, ok = data[h.prefix].(map[string]interface{})
		if !ok {
			return errors.Errorf("invalid data structure, missing prefix key: '%s'", h.prefix)
		}
	}

	// Load individual component settings
	for name, comp := range h.comp {
		data, ok := src[name]
		if ok {
			// nolint: forcetypeassert
			if err := comp.Restore(data.(map[string]interface{})); err != nil {
				return err
			}
		}
	}
	return nil
}
