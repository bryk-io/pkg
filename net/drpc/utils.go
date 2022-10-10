package drpc

import (
	"context"
	"crypto/tls"
	"os"

	"go.bryk.io/pkg/errors"
	"storj.io/drpc/drpcmetadata"
)

// Helper method to determine if a path exists and is a regular file.
func exists(name string) bool {
	info, err := os.Stat(name)
	return err == nil && !info.IsDir()
}

// LoadCertificate provides a helper method to conveniently parse and existing
// certificate and corresponding private key.
func LoadCertificate(cert []byte, key []byte) (tls.Certificate, error) {
	c, err := tls.X509KeyPair(cert, key)
	return c, errors.Wrap(err, "failed to load key pair")
}

// ContextWithMetadata adds custom data to a given context. This is particularly
// useful when sending outgoing requests on the client side.
func ContextWithMetadata(ctx context.Context, data map[string]string) context.Context {
	return drpcmetadata.AddPairs(ctx, data)
}

// MetadataFromContext retrieve any custom metadata available on the provided
// context. This is particularly useful when processing incoming requests on
// the server side.
func MetadataFromContext(ctx context.Context) (map[string]string, bool) {
	return drpcmetadata.Get(ctx)
}
