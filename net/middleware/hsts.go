package middleware

import (
	"bytes"
	"net/http"
	"strconv"
	"time"
)

const (
	// Minimum cache duration (in days) for inclusion in the Chrome HSTS list.
	// https://hstspreload.org/
	minimumPreloadAge = 365

	// Name of HTTP header to use when enforcing HSTS policy.
	httpHeaderKey = "Strict-Transport-Security"

	// Schema used for secure communication.
	scheme = "https"
)

// HSTS provides an HTTP Strict Transport Security implementation.
// When enabled this handler will redirect any HTTP request to its HTTPS
// representation while adding the required HSTS headers.
//
// Based on the original implementation: https://github.com/a-h/hsts
func HSTS(options HSTSOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			if isHTTPS(r, &options) {
				w.Header().Add(httpHeaderKey, header(&options))
				next.ServeHTTP(w, r)
				return
			}

			if options.HostOverride != "" {
				r.URL.Host = options.HostOverride
			} else if !r.URL.IsAbs() {
				r.URL.Host = r.Host
			}

			r.URL.Scheme = scheme
			http.Redirect(w, r, r.URL.String(), http.StatusMovedPermanently)
		}
		return http.HandlerFunc(fn)
	}
}

// HSTSOptions defines the configuration options available when enabling HSTS.
// nolint: lll
type HSTSOptions struct {
	// MaxAge sets the duration (in hours) that the HSTS is valid for.
	MaxAge uint `json:"max_age" yaml:"max_age" mapstructure:"max_age"`

	// HostOverride provides a host to the redirection URL in the case that
	// the system is behind a load balancer which doesn't provide the
	// X-Forwarded-Host HTTP header (e.g. an Amazon ELB).
	HostOverride string `json:"host_override" yaml:"host_override" mapstructure:"host_override"`

	// Decides whether to accept the X-Forwarded-Proto header as proof of SSL.
	AcceptXForwardedProtoHeader bool `json:"accept_forwarded_proto" yaml:"accept_forwarded_proto" mapstructure:"accept_forwarded_proto"`

	// SendPreloadDirective sets whether the preload directive should be set.
	// The directive allows browsers to confirm that the site should be added
	// to a preload list. https://hstspreload.org/
	SendPreloadDirective bool `json:"send_preload_directive" yaml:"send_preload_directive" mapstructure:"send_preload_directive"`

	// Whether to apply the HSTS policy to subdomains as well.
	IncludeSubdomains bool `json:"include_subdomains" yaml:"include_subdomains" mapstructure:"include_subdomains"`
}

// DefaultHSTSOptions return a sane default configuration to enable a HSTS policy.
func DefaultHSTSOptions() HSTSOptions {
	return HSTSOptions{
		MaxAge:                      24 * minimumPreloadAge,
		AcceptXForwardedProtoHeader: true,
		SendPreloadDirective:        false,
		IncludeSubdomains:           false,
	}
}

// Inspect if the provided HTTP request is a valid HTTPS request.
func isHTTPS(r *http.Request, options *HSTSOptions) bool {
	// Added by common load balancer which do TLS offloading
	if options.AcceptXForwardedProtoHeader && r.Header.Get("X-Forwarded-Proto") == scheme {
		return true
	}
	// If the X-Forwarded-Proto was set upstream as HTTP, then the request came in without TLS.
	if options.AcceptXForwardedProtoHeader && r.Header.Get("X-Forwarded-Proto") == "http" {
		return false
	}
	// Set by some middleware.
	if r.URL.Scheme == scheme {
		return true
	}
	// Set when the Go server is running HTTPS itself
	if r.TLS != nil && r.TLS.HandshakeComplete {
		return true
	}
	return false
}

// Get the HTTP header value for the handler instance.
func header(options *HSTSOptions) string {
	maxAge := time.Duration(options.MaxAge) * time.Hour
	buf := bytes.NewBufferString("max-age=")
	_, _ = buf.WriteString(strconv.Itoa(int(maxAge.Seconds())))
	if options.IncludeSubdomains {
		_, _ = buf.WriteString("; includeSubDomains")
	}
	if options.SendPreloadDirective {
		_, _ = buf.WriteString("; preload")
	}
	return buf.String()
}
