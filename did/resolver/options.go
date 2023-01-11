package resolver

// Option definitions provide a functional-style configuration
// mechanism for new resolver instances.
type Option func(i *Instance) error

// WithProvider registers/enables a DID method handler with the resolver
// instance.
func WithProvider(method string, prov Provider) Option {
	return func(i *Instance) error {
		i.providers[method] = prov
		return nil
	}
}

// WithEncoder registers/enables a DID document encoded with the resolver
// instance. The encoder `enc` will be responsible of production valid
// representations when requested by the user using `mime` data type.
// A default JSON encoder is registered by default for content types:
//   - application/ld+json
//   - application/did+ld+json
//   - application/ld+json;profile="https://w3id.org/did-resolution"
func WithEncoder(mime string, enc Encoder) Option {
	return func(i *Instance) error {
		i.encoders[mime] = enc
		return nil
	}
}
