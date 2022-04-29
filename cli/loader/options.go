package loader

// Option provides a functional-style mechanism to adjust internal settings
// when creating a new `Helper` instance.
type Option func(h *Helper) error

// WithPrefix allows to set a root prefix. If provided this prefix will
// be set on all cli.Params generated and data structures exported. This
// allows more flexibility over the portable data formats generated and
// consumed by the helper instance.
func WithPrefix(prefix string) Option {
	return func(h *Helper) error {
		h.prefix = prefix
		return nil
	}
}

// WithComponent registers a component in the helper instance.
func WithComponent(name string, comp Component) Option {
	return func(h *Helper) error {
		h.Register(name, comp)
		return nil
	}
}
