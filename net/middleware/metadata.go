package middleware

import (
	"context"
	"net/http"
	"strings"

	"google.golang.org/grpc/metadata"
)

// ContextMetadataOptions provide configuration settings available to
// adjust the behavior of the metadata middleware.
type ContextMetadataOptions struct {
	// The headers must be specified in its lowercase (non-canonical) form.
	// If no specific headers are provided, all headers in the request are
	// registered as metadata by default.
	Headers []string `json:"headers" yaml:"headers" mapstructure:"headers"`

	// Provides complete flexibility to adjust the metadata produced for a
	// received request.
	Hook func(md *Metadata, r http.Request) `json:"-" yaml:"-" mapstructure:"-"`
}

// ContextMetadata allows keeping HTTP headers or other request details as
// metadata in the context used when processing incoming requests. This
// allows other extensions and resolvers to have access to required information.
//
// Upstream elements can retrieve available metadata with:
//   md, ok := MetadataFromContext(ctx)
func ContextMetadata(options ContextMetadataOptions) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			md := Metadata{}
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

// Metadata provides a simple mechanism to propagate custom values
// through the context instance used while processing operations.
// Based on the gRPC implementation.
type Metadata = metadata.MD

// MetadataFromContext retrieves metadata information from the provided
// context if available.
func MetadataFromContext(ctx context.Context) (Metadata, bool) {
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
