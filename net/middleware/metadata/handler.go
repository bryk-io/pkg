package metadata

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// Options available to adjust the behavior of the metadata middleware.
type Options struct {
	// The headers must be specified in its lowercase (non-canonical) form.
	// If no specific headers are provided, all headers in the request are
	// registered as metadata by default.
	Headers []string `json:"headers" yaml:"headers" mapstructure:"headers"`

	// Provides complete flexibility to adjust the metadata produced for a
	// received request.
	Hook func(md *MD, r http.Request) `json:"-" yaml:"-" mapstructure:"-"`
}

// Handler allows keeping HTTP headers or other request details as
// metadata in the context used when processing incoming requests. This
// allows other extensions and resolvers to have access to required information.
//
// Upstream elements can retrieve available metadata with:
//
//	md, ok := FromContext(ctx)
func Handler(options Options) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			md := MD{}
			for hk, hv := range r.Header {
				if len(options.Headers) == 0 || contains(hk, options.Headers) {
					md.Set(hk, hv...)
				}
			}
			if options.Hook != nil {
				options.Hook(&md, *r)
			}
			ctx := metadata.NewIncomingContext(context.Background(), md)
			h.ServeHTTP(w, r.WithContext(ctx))
		}
		return http.HandlerFunc(fn)
	}
}

// MD provides a simple mechanism to propagate custom values
// through the context instance used while processing operations.
// Based on the gRPC implementation.
type MD = metadata.MD

// FromContext retrieves metadata information from the provided
// context if available.
func FromContext(ctx context.Context) (MD, bool) {
	return metadata.FromIncomingContext(ctx)
}

// Helper method to look for specific key in the provided list.
func contains(needle string, haystack []string) bool {
	for _, k := range haystack {
		if strings.EqualFold(k, needle) {
			return true
		}
	}
	return false
}
