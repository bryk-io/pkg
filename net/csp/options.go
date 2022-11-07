package csp

// Option elements provide a functional-style configuration system for
// CSP policies.
type Option func(p *Policy) error

// WithReportTo provides an endpoint where policy violations will be reported.
// Reports are submitted as JSON objects via POST requests to the provided URI.
// Multiple endpoints can be provided.
func WithReportTo(endpoint ...string) Option {
	return func(p *Policy) error {
		p.reportTo = append(p.reportTo, endpoint...)
		return nil
	}
}

// WithReportOnly will not affect the behavior of existing applications, but
// will still generate violation reports when patterns incompatible with CSP
// are detected, and send them to a reporting endpoint defined in your policy.
func WithReportOnly() Option {
	return func(p *Policy) error {
		p.reportOnly = true
		return nil
	}
}

// UnsafeEval allows the application to use the `eval()` JavaScript function.
// This reduces the protection against certain types of DOM-based XSS bugs,
// but makes it easier to adopt CSP. If your application doesn't use `eval()`,
// you can omit this option and have a safer policy.
func UnsafeEval() Option {
	return func(p *Policy) error {
		p.allowEval = true
		return nil
	}
}

// WithBaseURI restricts the URLs which can be used in a document's <base>
// element. If this value is absent, then any URI is allowed. If this directive
// is absent, the user agent will use the value in the <base> element.
func WithBaseURI(v string) Option {
	return func(p *Policy) error {
		p.baseURI = v
		return nil
	}
}

// WithDefaultSrc serves as a fallback for the other CSP fetch directives. For
// each fetch directive that is absent, the user agent looks for the `default-src`
// directive and use this value for it.
func WithDefaultSrc(v string) Option {
	return func(p *Policy) error {
		p.defaultSrc = v
		return nil
	}
}
