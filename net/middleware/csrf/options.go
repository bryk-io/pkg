package csrf

// Options available to adjust the behavior of the CSRF protection
// middleware.
type Options struct {
	// Allow requests coming from specific origins. The values listed here
	// must of the form "scheme://host[:port]".
	TrustedOrigins []string

	// Permits all requests that match the given pattern; this is considered
	// insecure.
	BypassPattern string
}
